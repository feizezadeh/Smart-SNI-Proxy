#!/bin/bash

echo "🔄 Deploying Smart SNI Proxy to server..."
echo ""

# Pull latest changes
echo "📥 Pulling latest changes from GitHub..."
cd ~/smartSNI
git pull origin main

echo ""
echo "🔨 Building application..."
GOROOT=/usr/local/go /usr/local/bin/go build -o smartSNI main.go

if [ $? -eq 0 ]; then
    echo "✅ Build successful"

    echo ""
    echo "🔄 Restarting sni service..."
    systemctl restart sni

    sleep 3

    echo ""
    echo "📊 Service Status:"
    systemctl status sni --no-pager | head -15

    echo ""
    echo "📝 Recent logs:"
    journalctl -u sni -n 10 --no-pager

    echo ""
    echo "✅ Deployment completed!"
    echo "🌐 Access web panel at: http://dns.dnsoverhttps.site:8088"
else
    echo "❌ Build failed"
    exit 1
fi
