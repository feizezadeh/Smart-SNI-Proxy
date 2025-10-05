#!/bin/bash

# Smart SNI Proxy v2.0 - Installation Script
# https://github.com/feizezadeh/Smart-SNI-Proxy

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
        git clone https://github.com/feizezadeh/Smart-SNI-Proxy.git /root/smartSNI

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
  "rate_limit_per_ip": 100,
  "rate_limit_burst_ip": 200,
  "log_level": "info",
  "trusted_proxies": [],
  "blocked_domains": [],
  "metrics_enabled": true,
  "web_panel_enabled": true,
  "web_panel_username": "admin",
  "web_panel_password": "$webpanel_pass_hash",
  "web_panel_port": 8088,
  "user_management": false
}
EOF
)

        # Save JSON to config.json file
        echo "$json_content" | jq '.' > /root/smartSNI/config.json

        nginx_conf="/etc/nginx/sites-enabled/default"
        sed -i "s/server_name _;/server_name $domain;/g" "$nginx_conf"
        sed -i "s/<YOUR_HOST>/$domain/g" /root/smartSNI/nginx.conf

        # Obtain SSL certificates for main domain
        certbot --nginx -d $domain --register-unsafely-without-email --non-interactive --agree-tos --redirect

        # Ask for DoH subdomain setup
        echo -e "${yellow}********************${rest}"
        read -p "Setup DoH subdomain? (y/n) [default: y]: " setup_doh
        setup_doh=${setup_doh:-y}

        if [[ "$setup_doh" == "y" || "$setup_doh" == "Y" ]]; then
            echo -e "${yellow}********************${rest}"
            read -p "Enter DoH subdomain (e.g., doh.$domain): " doh_domain
            doh_domain=${doh_domain:-doh.$domain}

            echo -e "${cyan}Creating DoH subdomain: $doh_domain${rest}"
            echo -e "${yellow}⚠️  Make sure DNS A record is set for $doh_domain${rest}"
            read -p "Press Enter when DNS is ready..."

            # Obtain SSL certificate for DoH subdomain
            systemctl stop nginx
            certbot certonly --standalone -d $doh_domain --register-unsafely-without-email --non-interactive --agree-tos

            # Create nginx config for DoH
            cat > /etc/nginx/sites-available/smartsni-doh <<EOF
upstream dohloop {
    zone dohloop 64k;
    server 127.0.0.1:8080;
    keepalive 32;
}

server {
    listen 8443 ssl http2;
    server_name $doh_domain;

    ssl_certificate /etc/letsencrypt/live/$doh_domain/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/$doh_domain/privkey.pem;
    include /etc/letsencrypt/options-ssl-nginx.conf;
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;

    # Security headers
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "no-referrer" always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload" always;

    # DoH endpoint
    location /dns-query {
        proxy_pass http://dohloop;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;

        proxy_connect_timeout 10s;
        proxy_send_timeout 10s;
        proxy_read_timeout 10s;

        proxy_buffering off;
        proxy_request_buffering off;
    }

    # Health check
    location /health {
        proxy_pass http://dohloop;
        proxy_http_version 1.1;
        access_log off;
    }
}
EOF

            # Enable DoH site
            ln -sf /etc/nginx/sites-available/smartsni-doh /etc/nginx/sites-enabled/

            # Open port 8443 in firewall (if ufw is active)
            if command -v ufw &> /dev/null && ufw status | grep -q "Status: active"; then
                ufw allow 8443/tcp
            fi
        fi

        sudo cp /root/smartSNI/nginx.conf "$nginx_conf"
        systemctl start nginx
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
            if [[ "$setup_doh" == "y" || "$setup_doh" == "Y" ]]; then
                echo -e "${green}  DoH (dedicated): ${cyan}https://$doh_domain:8443/dns-query${rest}"
            fi
            echo -e "${green}  DoT:  ${cyan}$domain:853${rest}"
            echo -e "${green}  SNI:  ${cyan}$domain:443${rest}"
            echo ""
            echo -e "${cyan}🌐 Web Panel:${rest}"
            echo -e "${green}  URL:      ${cyan}http://$myip:8088${rest}"
            echo -e "${green}  Username: ${cyan}admin${rest}"
            echo -e "${green}  Password: ${cyan}$webpanel_pass${rest}"
            echo -e "${yellow}  ⚠️  Save this password! It's only shown once.${rest}"
            echo ""
            echo -e "${cyan}👥 User Management:${rest}"
            echo -e "${green}  Status: ${cyan}Disabled (user_management: false)${rest}"
            echo -e "${yellow}  To enable: Set 'user_management: true' in config.json${rest}"
            echo -e "${green}  Features: ${cyan}IP-based access control with FIFO${rest}"
            echo ""
            echo -e "${cyan}📊 Monitoring:${rest}"
            echo -e "${green}  Health: ${cyan}http://127.0.0.1:8080/health${rest}"
            echo -e "${green}  Metrics: ${cyan}http://127.0.0.1:8080/metrics${rest}"
            echo ""
            echo -e "${cyan}📝 Logs:${rest}"
            echo -e "${green}  View: ${cyan}journalctl -u sni.service -f${rest}"
            echo ""
            echo -e "${cyan}📚 Documentation:${rest}"
            echo -e "${green}  DoH Usage: ${cyan}/root/smartSNI/DOH-USAGE.md${rest}"
            echo -e "${green}  DoH Setup: ${cyan}/root/smartSNI/DOH-SETUP-SUMMARY.md${rest}"
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
echo -e "${cyan}║  ${yellow}github.com/feizezadeh/Smart-SNI-Proxy${cyan}  ║${rest}"
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
echo -e "${purple}║ ${yellow}6${rest}] ${green}User Management${purple}       ║${rest}"
echo -e "${purple}║ ${yellow}7${rest}] ${green}View Logs${purple}             ║${rest}"
echo -e "${purple}║ ${yellow}8${rest}] ${green}View Metrics${purple}          ║${rest}"
echo -e "${purple}║ ${yellow}9${rest}] ${green}Restart Service${purple}       ║${rest}"
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

# User Management Menu
user_management() {
    if [ ! -f "/etc/systemd/system/sni.service" ]; then
        echo -e "${yellow}********************${rest}"
        echo -e "${red}Service is not installed.${rest}"
        echo -e "${yellow}********************${rest}"
        return
    fi

    config_file="/root/smartSNI/config.json"
    user_mgmt_enabled=$(jq -r '.user_management' "$config_file" 2>/dev/null)

    clear
    echo -e "${cyan}╔══════════════════════════╗${rest}"
    echo -e "${cyan}║  ${green}User Management Menu${cyan}  ║${rest}"
    echo -e "${cyan}╚══════════════════════════╝${rest}"
    echo ""
    echo -e "${cyan}Status: ${rest}"
    if [[ "$user_mgmt_enabled" == "true" ]]; then
        echo -e "${green}✅ Enabled${rest}"
    else
        echo -e "${yellow}⚠️  Disabled${rest}"
    fi
    echo ""
    echo -e "${purple}╔════════════════════════════╗${rest}"
    echo -e "${purple}║ ${yellow}1${rest}] ${green}Enable User Management${purple}  ║${rest}"
    echo -e "${purple}║ ${yellow}2${rest}] ${green}Disable User Management${purple} ║${rest}"
    echo -e "${purple}║ ${yellow}3${rest}] ${green}Open Web Panel${purple}          ║${rest}"
    echo -e "${purple}║ ${yellow}4${rest}] ${green}View Panel Info${purple}         ║${rest}"
    echo -e "${purple}║ ${red}0${rest}] ${purple}Back to Main Menu${purple}       ║${rest}"
    echo -e "${purple}╚════════════════════════════╝${rest}"
    echo ""

    read -p "Enter your choice: " um_choice

    case "$um_choice" in
        1)
            jq '.user_management = true' "$config_file" > temp_config.json
            mv temp_config.json "$config_file"
            echo -e "${green}✅ User Management enabled!${rest}"
            echo -e "${yellow}Restarting service...${rest}"
            systemctl restart sni.service
            sleep 2
            echo -e "${green}Done! Users now need to register their IPs to access services.${rest}"
            echo -e "${cyan}Use Web Panel to create users at: http://$myip:8088${rest}"
            ;;
        2)
            jq '.user_management = false' "$config_file" > temp_config.json
            mv temp_config.json "$config_file"
            echo -e "${yellow}⚠️  User Management disabled!${rest}"
            echo -e "${yellow}Restarting service...${rest}"
            systemctl restart sni.service
            sleep 2
            echo -e "${green}Done! All IPs now have access to services.${rest}"
            ;;
        3)
            web_port=$(jq -r '.web_panel_port' "$config_file" 2>/dev/null)
            echo -e "${cyan}Opening Web Panel...${rest}"
            echo -e "${green}URL: http://$myip:$web_port${rest}"
            echo -e "${yellow}Default username: admin${rest}"
            echo -e "${yellow}⚠️  Password was shown during installation${rest}"
            ;;
        4)
            web_port=$(jq -r '.web_panel_port' "$config_file" 2>/dev/null)
            web_enabled=$(jq -r '.web_panel_enabled' "$config_file" 2>/dev/null)
            echo -e "${cyan}╔═══════════════════════════╗${rest}"
            echo -e "${cyan}║  ${green}Web Panel Information${cyan}  ║${rest}"
            echo -e "${cyan}╚═══════════════════════════╝${rest}"
            echo ""
            echo -e "${green}Status: ${rest}"
            if [[ "$web_enabled" == "true" ]]; then
                echo -e "${green}✅ Enabled${rest}"
            else
                echo -e "${red}❌ Disabled${rest}"
            fi
            echo ""
            echo -e "${green}URL:${rest} ${cyan}http://$myip:$web_port${rest}"
            echo -e "${green}Username:${rest} ${cyan}admin${rest}"
            echo -e "${yellow}Password: Check installation output${rest}"
            echo ""
            echo -e "${cyan}Features:${rest}"
            echo -e "  ${green}•${rest} Create users with expiration dates"
            echo -e "  ${green}•${rest} Set max IPs per user (FIFO replacement)"
            echo -e "  ${green}•${rest} Generate registration links for users"
            echo -e "  ${green}•${rest} View user statistics and IP lists"
            echo -e "  ${green}•${rest} Enable/disable users"
            echo ""
            ;;
        0)
            return
            ;;
        *)
            echo -e "${red}Invalid choice${rest}"
            ;;
    esac

    echo ""
    read -p "Press Enter to continue..."
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
        user_management
        ;;
    7)
        view_logs
        ;;
    8)
        view_metrics
        ;;
    9)
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
