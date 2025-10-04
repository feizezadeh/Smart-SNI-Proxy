#!/bin/bash

# This script will be uploaded and run on the server

cd ~/smartSNI || exit 1

echo "📝 Updating git remote URL..."
git remote set-url origin https://github.com/feizezadeh/Smart-SNI-Proxy.git

echo "⬇️ Pulling latest changes..."
git fetch origin
git reset --hard origin/main

echo "🔨 Building application..."
/usr/local/go/bin/go build -o smartSNI main.go

if [ $? -eq 0 ]; then
    echo "✅ Build successful"

    echo "🔄 Restarting sni service..."
    systemctl restart sni

    sleep 2

    echo ""
    echo "📊 Service Status:"
    systemctl status sni --no-pager | head -20

    echo ""
    echo "✅ Update completed!"
    echo "🌐 Access web panel at: http://dns.dnsoverhttps.site:8088"
else
    echo "❌ Build failed"
    exit 1
fi
