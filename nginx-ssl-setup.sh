#!/bin/bash

# Setup SSL for Smart SNI Proxy Web Panel using Nginx

DOMAIN="dns.dnsoverhttps.site"
EMAIL="your-email@example.com"  # Change this to your email

echo "🔒 Setting up SSL for Smart SNI Proxy Web Panel..."
echo ""

# Create nginx config for web panel
cat > /etc/nginx/sites-available/smartsni <<EOF
server {
    listen 80;
    server_name ${DOMAIN};

    location / {
        return 301 https://\$server_name\$request_uri;
    }
}

server {
    listen 443 ssl http2;
    server_name ${DOMAIN};

    # SSL certificates (will be created by certbot)
    ssl_certificate /etc/letsencrypt/live/${DOMAIN}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/${DOMAIN}/privkey.pem;

    # SSL settings
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # Proxy settings
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

# Enable the site
ln -sf /etc/nginx/sites-available/smartsni /etc/nginx/sites-enabled/

# Test nginx config
echo "📝 Testing nginx configuration..."
nginx -t

if [ $? -eq 0 ]; then
    echo "✅ Nginx config is valid"

    # Get SSL certificate
    echo ""
    echo "📜 Obtaining SSL certificate from Let's Encrypt..."
    certbot certonly --nginx -d ${DOMAIN} --non-interactive --agree-tos -m ${EMAIL}

    if [ $? -eq 0 ]; then
        echo "✅ SSL certificate obtained successfully"

        # Reload nginx
        echo ""
        echo "🔄 Reloading nginx..."
        systemctl reload nginx

        echo ""
        echo "✅ Setup completed!"
        echo ""
        echo "🌐 Access web panel at: https://${DOMAIN}"
        echo ""
        echo "📝 Note: Panel is now accessible via HTTPS on port 443"
        echo "         HTTP (port 80) will redirect to HTTPS"
    else
        echo "❌ Failed to obtain SSL certificate"
        exit 1
    fi
else
    echo "❌ Nginx config test failed"
    exit 1
fi
