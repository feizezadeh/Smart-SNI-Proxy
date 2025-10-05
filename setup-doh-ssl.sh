#!/bin/bash

# Setup DoH endpoint with SSL on subdomain

DOMAIN="dns.dnsoverhttps.site"
DOH_SUBDOMAIN="doh.dnsoverhttps.site"  # Alternative: use main domain

echo "🔒 Setting up DoH SSL endpoint..."
echo ""

# Check if using subdomain or main domain
read -p "Use subdomain 'doh.dnsoverhttps.site' or main domain? (subdomain/main) [main]: " choice
choice=${choice:-main}

if [ "$choice" = "subdomain" ]; then
    DOH_DOMAIN="$DOH_SUBDOMAIN"
else
    DOH_DOMAIN="$DOMAIN"
fi

echo "Using domain: $DOH_DOMAIN"
echo ""

# Check if certificate exists
if [ ! -f "/etc/letsencrypt/live/${DOH_DOMAIN}/fullchain.pem" ]; then
    echo "📜 Obtaining SSL certificate for ${DOH_DOMAIN}..."

    # Stop nginx temporarily to free port 80
    systemctl stop nginx

    certbot certonly --standalone --preferred-challenges http \
        -d ${DOH_DOMAIN} --non-interactive --agree-tos \
        -m admin@${DOH_DOMAIN}

    if [ $? -ne 0 ]; then
        echo "❌ Failed to obtain certificate"
        systemctl start nginx
        exit 1
    fi

    systemctl start nginx
fi

# Create nginx config for DoH
cat > /etc/nginx/sites-available/smartsni-doh <<EOF
# DoH endpoint on port 443
server {
    listen 443 ssl http2;
    server_name ${DOH_DOMAIN};

    # SSL certificates
    ssl_certificate /etc/letsencrypt/live/${DOH_DOMAIN}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/${DOH_DOMAIN}/privkey.pem;

    # SSL settings
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;

    # DoH endpoint
    location /dns-query {
        proxy_pass http://127.0.0.1:8080/dns-query;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;

        # DoH specific headers
        proxy_set_header Accept application/dns-message;
        proxy_set_header Content-Type application/dns-message;

        # Disable buffering for DoH
        proxy_buffering off;
        proxy_request_buffering off;
    }

    # Health check endpoint
    location /health {
        proxy_pass http://127.0.0.1:8080/health;
        proxy_set_header Host \$host;
    }

    # Redirect root to info page
    location = / {
        return 200 'DoH Server - Use: https://${DOH_DOMAIN}/dns-query\n';
        add_header Content-Type text/plain;
    }
}

# HTTP redirect
server {
    listen 80;
    server_name ${DOH_DOMAIN};
    return 301 https://\$server_name\$request_uri;
}
EOF

# Enable the site
ln -sf /etc/nginx/sites-available/smartsni-doh /etc/nginx/sites-enabled/

# Test nginx config
echo "📝 Testing nginx configuration..."
nginx -t

if [ $? -eq 0 ]; then
    echo "✅ Nginx config is valid"

    # Reload nginx
    echo "🔄 Reloading nginx..."
    systemctl reload nginx

    echo ""
    echo "✅ DoH SSL setup completed!"
    echo ""
    echo "🌐 DoH Endpoint: https://${DOH_DOMAIN}/dns-query"
    echo ""
    echo "📝 Test with:"
    echo "   curl -H 'accept: application/dns-json' 'https://${DOH_DOMAIN}/dns-query?name=google.com&type=A'"
    echo ""
    echo "⚠️  IMPORTANT:"
    echo "   - DoH is now accessible via HTTPS on port 443"
    echo "   - Configure your browser/system to use: https://${DOH_DOMAIN}/dns-query"
else
    echo "❌ Nginx config test failed"
    exit 1
fi
