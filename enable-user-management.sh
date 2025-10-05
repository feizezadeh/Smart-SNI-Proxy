#!/bin/bash

# Enable User Management in Smart SNI Proxy

echo "🔐 Enabling User Management..."
echo ""

cd ~/smartSNI

# Backup config
cp config.json config.json.backup
echo "✅ Config backed up to config.json.backup"

# Enable user_management in config.json
sed -i 's/"user_management": false/"user_management": true/' config.json

echo "✅ User management enabled in config.json"

# Restart service
echo ""
echo "🔄 Restarting sni service..."
systemctl restart sni

sleep 2

if systemctl is-active --quiet sni; then
    echo "✅ Service restarted successfully"
    echo ""
    echo "📋 User Management is now ACTIVE"
    echo ""
    echo "⚠️  IMPORTANT:"
    echo "   - Only users with registered IPs can access DNS services"
    echo "   - Create users from web panel: ➕ Create User"
    echo "   - Each user gets a unique registration link"
    echo "   - Users can register up to MaxIPs (FIFO)"
    echo ""
    echo "🌐 Web Panel: http://$(hostname -I | awk '{print $1}'):8088"
else
    echo "❌ Service failed to start"
    echo "   Restoring backup..."
    mv config.json.backup config.json
    systemctl restart sni
    exit 1
fi
