#!/bin/bash

# Setup SSL for Web Panel on port 8443 (since 443 is used by SNI proxy)

DOMAIN="dns.dnsoverhttps.site"

echo "🔒 Setting up SSL for Web Panel on port 8443..."
echo ""

# Check if certificate exists
if [ ! -f "/etc/letsencrypt/live/${DOMAIN}/fullchain.pem" ]; then
    echo "📜 Obtaining SSL certificate..."
    certbot certonly --standalone --preferred-challenges http -d ${DOMAIN} --non-interactive --agree-tos -m admin@${DOMAIN}

    if [ $? -ne 0 ]; then
        echo "❌ Failed to obtain certificate"
        echo "   Make sure port 80 is accessible and DNS is pointing to this server"
        exit 1
    fi
fi

# Create nginx config for web panel on port 8443
cat > /etc/nginx/sites-available/smartsni-panel <<EOF
server {
    listen 8443 ssl http2;
    server_name ${DOMAIN};

    # SSL certificates
    ssl_certificate /etc/letsencrypt/live/${DOMAIN}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/${DOMAIN}/privkey.pem;

    # SSL settings
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    # Proxy to local web panel
    location / {
        proxy_pass http://127.0.0.1:8088;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_cache_bypass \$http_upgrade;
    }
}
EOF

# Remove old config if exists
rm -f /etc/nginx/sites-enabled/smartsni

# Enable the site
ln -sf /etc/nginx/sites-available/smartsni-panel /etc/nginx/sites-enabled/

# Test nginx config
echo "📝 Testing nginx configuration..."
nginx -t

if [ $? -eq 0 ]; then
    echo "✅ Nginx config is valid"

    # Reload nginx
    echo "🔄 Reloading nginx..."
    systemctl reload nginx

    echo ""
    echo "✅ Setup completed!"
    echo ""
    echo "🌐 Access web panel at: https://${DOMAIN}:8443"
    echo ""
    echo "📝 Notes:"
    echo "   - Web panel is accessible on HTTPS port 8443"
    echo "   - Port 443 is used by SNI Proxy (unchanged)"
    echo "   - Make sure firewall allows port 8443"
else
    echo "❌ Nginx config test failed"
    exit 1
fi
