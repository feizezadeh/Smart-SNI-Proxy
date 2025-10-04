#!/bin/bash

echo "🔄 Updating Smart SNI Proxy on server..."
echo ""

# Change to smartSNI directory
cd ~/smartSNI || exit 1

# Update git remote URL if needed
CURRENT_REMOTE=$(git remote get-url origin)
if [[ "$CURRENT_REMOTE" != *"feizezadeh/Smart-SNI-Proxy"* ]]; then
    echo "📝 Updating git remote URL..."
    git remote set-url origin https://github.com/feizezadeh/Smart-SNI-Proxy.git
fi

# Pull latest changes
echo "⬇️ Pulling latest changes..."
git fetch origin
git reset --hard origin/main

# Build the application
echo "🔨 Building application..."
/usr/local/go/bin/go build -o smartSNI main.go

if [ $? -eq 0 ]; then
    echo "✅ Build successful"

    # Restart the sni service (not smartsni)
    echo "🔄 Restarting sni service..."
    sudo systemctl restart sni

    echo ""
    echo "⏳ Waiting for service to start..."
    sleep 3

    # Check service status
    echo ""
    echo "📊 Service Status:"
    sudo systemctl status sni --no-pager -l | head -15

    echo ""
    echo "📝 Recent logs:"
    sudo journalctl -u sni -n 20 --no-pager

    echo ""
    echo "🌐 Web Panel Access:"
    echo "   HTTP: http://dns.dnsoverhttps.site:8088"
    echo "   (Note: Use HTTP not HTTPS for web panel)"

else
    echo "❌ Build failed"
    exit 1
fi
