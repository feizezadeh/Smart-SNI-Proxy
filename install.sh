#!/bin/bash

# Smart SNI Proxy v2.0 - Installation Script
# https://github.com/Ptechgithub/smartSNI

#colors
red='\033[0;31m'
green='\033[0;32m'
yellow='\033[0;33m'
blue='\033[0;34m'
purple='\033[0;35m'
cyan='\033[0;36m'
rest='\033[0m'
myip=$(hostname -I | awk '{print $1}')

# Version
VERSION="2.0"

# Function to detect Linux distribution
detect_distribution() {
    local supported_distributions=("ubuntu" "debian" "centos" "fedora")
    
    if [ -f /etc/os-release ]; then
        source /etc/os-release
        if [[ "${ID}" = "ubuntu" || "${ID}" = "debian" || "${ID}" = "centos" || "${ID}" = "fedora" ]]; then
            pm="apt"
            [ "${ID}" = "centos" ] && pm="yum"
            [ "${ID}" = "fedora" ] && pm="dnf"
        else
            echo "Unsupported distribution!"
            exit 1
        fi
    else
        echo "Unsupported distribution!"
        exit 1
    fi
}

# Install necessary packages
install_dependencies() {
    detect_distribution
    $pm update -y
    local packages=("nginx" "git" "jq" "certbot" "python3-certbot-nginx" "wget" "tar")
    
    for package in "${packages[@]}"; do
        if ! dpkg -s "$package" &> /dev/null; then
            echo -e "${yellow}$package is not installed. Installing...${rest}"
            $pm install -y "$package"
        else
            echo -e "${green}$package is already installed.${rest}"
        fi
    done
    
    if ! command -v go &> /dev/null; then
        install_go
    else
        echo -e "${green}go is already installed.${rest}"
    fi
}

# Install Go
install_go() {
    echo -e "${yellow}go is not installed. Installing...${rest}"
    
    ARCH=$(dpkg --print-architecture)
    
    if [[ $ARCH == "amd64" || $ARCH == "arm64" ]]; then
        wget https://go.dev/dl/go1.21.1.linux-"$ARCH".tar.gz
        rm -rf /usr/local/go && rm -rf /usr/local/bin/go && tar -C /usr/local -xzf go1.21.1.linux-"$ARCH".tar.gz
        export PATH=$PATH:/usr/local/go/bin
        cp /usr/local/go/bin/go /usr/local/bin
        
        rm go1.21.1.linux-"$ARCH".tar.gz
        rm -rf /root/go
        echo -e "${cyan}Go has been installed.${rest}"
    else
        echo -e "${red}Unsupported architecture: $ARCH${rest}"
        exit 1
    fi
}

# install SNI service
install() {
    if systemctl is-active --quiet sni.service; then
        echo -e "${yellow}********************${rest}"
        echo -e "${green}Service is already installed and active.${rest}"
        echo -e "${yellow}********************${rest}"
    else
        install_dependencies
        git clone https://github.com/Ptechgithub/smartSNI.git /root/smartSNI
         
        sleep 1
        clear
        echo -e "${yellow}********************${rest}"
        read -p "Enter your domain: " domain
        echo -e "${yellow}********************${rest}"
        read -p "Enter Website names (separated by commas)[example: intel.com,youtube]: " site_list
        echo -e "${yellow}********************${rest}"
        # Split the input into an array
        IFS=',' read -ra sites <<< "$site_list"
        
        # Prepare a string with the new domains (with wildcard support)
        new_domains="{"
        for ((i = 0; i < ${#sites[@]}; i++)); do
            # Add both exact and wildcard patterns
            new_domains+="\"${sites[i]}\": \"$myip\", \"*.${sites[i]}\": \"$myip\""
            if [ $i -lt $((${#sites[@]}-1)) ]; then
                new_domains+=", "
            fi
        done
        new_domains+="}"

        # Generate random password for web panel
        webpanel_pass=$(openssl rand -hex 16)
        webpanel_pass_hash=$(echo -n "$webpanel_pass" | sha256sum | awk '{print $1}')

        # Create a JSON Object with host, domains and v2.0 settings
        json_content=$(cat <<EOF
{
  "host": "$domain",
  "domains": $new_domains,
  "upstream_doh": [
    "https://1.1.1.1/dns-query",
    "https://1.0.0.1/dns-query",
    "https://8.8.8.8/dns-query"
  ],
  "enable_auth": false,
  "auth_tokens": [],
  "cache_ttl": 300,
  "rate_limit_per_ip": 10,
  "rate_limit_burst_ip": 20,
  "log_level": "info",
  "trusted_proxies": [],
  "blocked_domains": [],
  "metrics_enabled": true,
  "web_panel_enabled": true,
  "web_panel_username": "admin",
  "web_panel_password": "$webpanel_pass_hash",
  "web_panel_port": 8088
}
EOF
)

        # Save JSON to config.json file
        echo "$json_content" | jq '.' > /root/smartSNI/config.json

        nginx_conf="/etc/nginx/sites-enabled/default"
        sed -i "s/server_name _;/server_name $domain;/g" "$nginx_conf"
        sed -i "s/<YOUR_HOST>/$domain/g" /root/smartSNI/nginx.conf

        # Obtain SSL certificates
        certbot --nginx -d $domain --register-unsafely-without-email --non-interactive --agree-tos --redirect

        sudo cp /root/smartSNI/nginx.conf "$nginx_conf"
        systemctl stop nginx
        systemctl restart nginx

        config_file="/root/smartSNI/config.json"

        sed -i "s/<YOUR_HOST>/$domain/g" "$config_file"
        sed -i "s/<YOUR_IP>/$myip/g" "$config_file"
        
        # Build the binary
        echo -e "${yellow}********************${rest}"
        echo -e "${cyan}Building smartSNI v$VERSION...${rest}"
        cd /root/smartSNI
        /usr/local/go/bin/go build -o smartsni main.go

        if [ $? -ne 0 ]; then
            echo -e "${red}Build failed! Please check the errors above.${rest}"
            exit 1
        fi
        echo -e "${green}Build successful!${rest}"

        # Create systemd service file
        cat > /etc/systemd/system/sni.service <<EOL
[Unit]
Description=Smart SNI Proxy v$VERSION
After=network.target

[Service]
User=root
WorkingDirectory=/root/smartSNI
ExecStart=/root/smartSNI/smartsni
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

# Security settings
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOL

        # Reload systemd, enable and start the service
        systemctl daemon-reload
        systemctl enable sni.service
        systemctl start sni.service

        # Wait a moment for service to start
        sleep 2

        # Check if the service is active
        if systemctl is-active --quiet sni.service; then
            echo -e "${yellow}_______________________________________${rest}"
            echo -e "${green}Smart SNI v$VERSION Installed Successfully!${rest}"
            echo -e "${yellow}_______________________________________${rest}"
            echo ""
            echo -e "${cyan}📡 Endpoints:${rest}"
            echo -e "${green}  DoH:  ${cyan}https://$domain/dns-query${rest}"
            echo -e "${green}  DoT:  ${cyan}$domain:853${rest}"
            echo -e "${green}  SNI:  ${cyan}$domain:443${rest}"
            echo ""
            echo -e "${cyan}🌐 Web Panel:${rest}"
            echo -e "${green}  URL:      ${cyan}http://$myip:8088${rest}"
            echo -e "${green}  Username: ${cyan}admin${rest}"
            echo -e "${green}  Password: ${cyan}$webpanel_pass${rest}"
            echo -e "${yellow}  ⚠️  Save this password! It's only shown once.${rest}"
            echo ""
            echo -e "${cyan}📊 Monitoring:${rest}"
            echo -e "${green}  Health: ${cyan}http://127.0.0.1:8080/health${rest}"
            echo -e "${green}  Metrics: ${cyan}http://127.0.0.1:8080/metrics${rest}"
            echo ""
            echo -e "${cyan}📝 Logs:${rest}"
            echo -e "${green}  View: ${cyan}journalctl -u sni.service -f${rest}"
            echo -e "${yellow}_______________________________________${rest}"
        else
            echo -e "${yellow}____________________________${rest}"
            echo -e "${red}Service is not active.${rest}"
            echo -e "${cyan}Check logs: journalctl -u sni.service -n 50${rest}"
            echo -e "${yellow}____________________________${rest}"
        fi
    fi
}

# Uninstall function
uninstall() {
    if [ ! -f "/etc/systemd/system/sni.service" ]; then
        echo -e "${yellow}____________________________${rest}"
        echo -e "${red}The service is not installed.${rest}"
        echo -e "${yellow}____________________________${rest}"
        return
    fi
    # Stop and disable the service
    sudo systemctl stop sni.service
    sudo systemctl disable sni.service 2>/dev/null

    # Remove service file
    sudo rm /etc/systemd/system/sni.service
    rm -rf /root/smartSNI
    rm -rf /root/go
    echo -e "${yellow}____________________________________${rest}"
    echo -e "${green}Uninstallation completed successfully.${rest}"
    echo -e "${yellow}____________________________________${rest}"
}

# Show Websites
display_sites() {
    config_file="/root/smartSNI/config.json"

    if [ -d "/root/smartSNI" ]; then
        echo -e "${yellow}****${cyan} [Websites] ${yellow}****${rest}"
        # Initialize a counter
        counter=1
        # Loop through the domains and display with numbering
        jq -r '.domains | keys_unsorted | .[]' "$config_file" | while read -r domain; do
            echo "$counter) $domain"
            ((counter++))
        done
        echo ""
        echo -e "${yellow}********************${rest}"
    else
        echo -e "${yellow}********************${rest}"
        echo -e "${red}Not installed. Please Install first.${rest}"
    fi
}

# Check service
check() {
    if systemctl is-active --quiet sni.service; then
        echo -e "${cyan}[Service Actived]${rest}"
    else
        echo -e "${yellow}[Service Not Active]${rest}"
    fi
}

# Add sites
add_sites() {
    config_file="/root/smartSNI/config.json"

    if [ -d "/root/smartSNI" ]; then
        echo -e "${yellow}********************${rest}"
        read -p "Enter additional Websites (separated by commas): " additional_sites
        IFS=',' read -ra new_sites <<< "$additional_sites"

        current_domains=$(jq -r '.domains | keys_unsorted | .[]' "$config_file")
        for site in "${new_sites[@]}"; do
            site=$(echo "$site" | xargs)  # Trim whitespace
            site_pattern="*.${site}"

            # Add both exact and wildcard patterns
            if [[ ! " ${current_domains[@]} " =~ " $site " ]]; then
                jq ".domains += {\"$site\": \"$myip\", \"$site_pattern\": \"$myip\"}" "$config_file" > temp_config.json
                mv temp_config.json "$config_file"
                echo -e "${yellow}********************${rest}"
                echo -e "${green}Domains ${cyan}'$site'${green} and ${cyan}'$site_pattern'${green} added successfully.${rest}"
            else
                echo -e "${yellow}Domain ${cyan}'$site' already exists.${rest}"
            fi
        done

        echo -e "${yellow}********************${rest}"
        echo -e "${cyan}Reloading configuration...${rest}"

        # Try hot reload first
        reload_response=$(curl -s -X POST http://127.0.0.1:8080/admin/reload 2>&1)

        if [ $? -eq 0 ]; then
            echo -e "${green}Configuration reloaded successfully (no restart needed)!${rest}"
        else
            # Fallback to service restart
            echo -e "${yellow}Hot reload failed, restarting service...${rest}"
            systemctl restart sni.service
            echo -e "${green}Service restarted successfully!${rest}"
        fi
    else
        echo -e "${yellow}********************${rest}"
        echo -e "${red}Not installed. Please Install first.${rest}"
    fi
}

# Remove sites
remove_sites() {
    config_file="/root/smartSNI/config.json"

    if [ -d "/root/smartSNI" ]; then
        # Display available sites
        display_sites

        read -p "Enter domain patterns to remove (separated by commas): " domains_to_remove
        IFS=',' read -ra selected_domains <<< "$domains_to_remove"

        # Remove selected domains from JSON
        for selected_domain in "${selected_domains[@]}"; do
            # Try with wildcard pattern first
            if jq -e --arg selected_domain "*.${selected_domain}" '.domains | has($selected_domain)' "$config_file" > /dev/null; then
                jq "del(.domains[\"*.${selected_domain}\"])" "$config_file" > temp_config.json
                mv temp_config.json "$config_file"
                echo -e "${yellow}********************${rest}"
                echo -e "${green}Domain ${cyan}'*.${selected_domain}'${green} removed successfully.${rest}"
            elif jq -e --arg selected_domain "$selected_domain" '.domains | has($selected_domain)' "$config_file" > /dev/null; then
                jq "del(.domains[\"$selected_domain\"])" "$config_file" > temp_config.json
                mv temp_config.json "$config_file"
                echo -e "${yellow}********************${rest}"
                echo -e "${green}Domain ${cyan}'$selected_domain'${green} removed successfully.${rest}"
            else
                echo -e "${yellow}********************${rest}"
                echo -e "${yellow}Domain ${cyan}'$selected_domain'${yellow} not found.${rest}"
            fi
        done

        echo -e "${yellow}********************${rest}"
        echo -e "${cyan}Reloading configuration...${rest}"

        # Try hot reload first
        reload_response=$(curl -s -X POST http://127.0.0.1:8080/admin/reload 2>&1)

        if [ $? -eq 0 ]; then
            echo -e "${green}Configuration reloaded successfully (no restart needed)!${rest}"
        else
            # Fallback to service restart
            echo -e "${yellow}Hot reload failed, restarting service...${rest}"
            systemctl restart sni.service
            echo -e "${green}Service restarted successfully!${rest}"
        fi
    else
        echo -e "${yellow}********************${rest}"
        echo -e "${red}Not installed. Please Install first.${rest}"
    fi
}

clear
echo -e "${cyan}╔══════════════════════════════════════════╗${rest}"
echo -e "${cyan}║  ${green}Smart SNI Proxy v${VERSION}${cyan}                 ║${rest}"
echo -e "${cyan}║  ${yellow}By Peyman - github.com/Ptechgithub${cyan}  ║${rest}"
echo -e "${cyan}╚══════════════════════════════════════════╝${rest}"
echo ""
check
echo ""
echo -e "${purple}╔════════════════════════════╗${rest}"
echo -e "${purple}║ ${green}      MAIN MENU${purple}          ║${rest}"
echo -e "${purple}╠════════════════════════════╣${rest}"
echo -e "${purple}║ ${yellow}1${rest}] ${green}Install / Upgrade${purple}     ║${rest}"
echo -e "${purple}║ ${yellow}2${rest}] ${green}Uninstall${purple}             ║${rest}"
echo -e "${purple}║ ${yellow}3${rest}] ${green}Show Websites${purple}         ║${rest}"
echo -e "${purple}║ ${yellow}4${rest}] ${green}Add Sites${purple}             ║${rest}"
echo -e "${purple}║ ${yellow}5${rest}] ${green}Remove Sites${purple}          ║${rest}"
echo -e "${purple}║ ${yellow}6${rest}] ${green}View Logs${purple}             ║${rest}"
echo -e "${purple}║ ${yellow}7${rest}] ${green}View Metrics${purple}          ║${rest}"
echo -e "${purple}║ ${yellow}8${rest}] ${green}Restart Service${purple}       ║${rest}"
echo -e "${purple}║ ${red}0${rest}] ${purple}Exit${purple}                  ║${rest}"
echo -e "${purple}╚════════════════════════════╝${rest}"
echo ""
# View logs
view_logs() {
    if [ ! -f "/etc/systemd/system/sni.service" ]; then
        echo -e "${yellow}********************${rest}"
        echo -e "${red}Service is not installed.${rest}"
        echo -e "${yellow}********************${rest}"
        return
    fi
    echo -e "${cyan}Showing last 50 log entries (Ctrl+C to exit)...${rest}"
    echo ""
    journalctl -u sni.service -n 50 --no-pager
    echo ""
    echo -e "${yellow}To follow live logs, use: ${cyan}journalctl -u sni.service -f${rest}"
}

# View metrics
view_metrics() {
    if ! systemctl is-active --quiet sni.service; then
        echo -e "${yellow}********************${rest}"
        echo -e "${red}Service is not running.${rest}"
        echo -e "${yellow}********************${rest}"
        return
    fi

    echo -e "${cyan}Fetching metrics...${rest}"
    echo ""

    metrics=$(curl -s http://127.0.0.1:8080/metrics 2>/dev/null)

    if [ $? -eq 0 ] && [ -n "$metrics" ]; then
        echo -e "${green}=== Service Metrics ===${rest}"
        echo "$metrics" | jq '.' 2>/dev/null || echo "$metrics"
        echo ""

        health=$(curl -s http://127.0.0.1:8080/health 2>/dev/null)
        if [ $? -eq 0 ] && [ -n "$health" ]; then
            echo -e "${green}=== Health Status ===${rest}"
            echo "$health" | jq '.' 2>/dev/null || echo "$health"
        fi
    else
        echo -e "${red}Failed to fetch metrics. Is the service running?${rest}"
    fi
    echo ""
}

# Restart service
restart_service() {
    if [ ! -f "/etc/systemd/system/sni.service" ]; then
        echo -e "${yellow}********************${rest}"
        echo -e "${red}Service is not installed.${rest}"
        echo -e "${yellow}********************${rest}"
        return
    fi

    echo -e "${cyan}Restarting Smart SNI service...${rest}"
    systemctl restart sni.service
    sleep 2

    if systemctl is-active --quiet sni.service; then
        echo -e "${green}Service restarted successfully!${rest}"
    else
        echo -e "${red}Failed to restart service. Check logs for details.${rest}"
    fi
}

read -p "Enter your choice: " choice
case "$choice" in
    1)
        install
        ;;
    2)
        uninstall
        ;;
    3)
        display_sites
        ;;
    4)
        add_sites
        ;;
    5)
        remove_sites
        ;;
    6)
        view_logs
        ;;
    7)
        view_metrics
        ;;
    8)
        restart_service
        ;;
    0)
        echo -e "${cyan}Goodbye! 👋${rest}"
        exit
        ;;
    *)
        echo -e "${yellow}********************${rest}"
        echo "Invalid choice. Please select a valid option."
        ;;
esac
