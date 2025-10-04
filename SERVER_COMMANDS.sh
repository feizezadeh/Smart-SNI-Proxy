#!/bin/bash
# این دستورات را مستقیماً روی سرور اجرا کنید
# SSH: ssh root@89.47.113.135

echo "🔄 Updating Smart SNI Proxy..."

# رفتن به دایرکتوری
cd /root/smartSNI

# Backup فایل‌های قدیمی
cp main.go main.go.backup
cp webpanel.html webpanel.html.backup
cp config.json config.json.backup

# دریافت آخرین تغییرات از GitHub
git fetch origin
git reset --hard origin/main

# نمایش تغییرات
echo "📋 Changes pulled from GitHub:"
git log -1 --oneline

# Fix config placeholders
echo "🔧 Fixing config.json..."
SERVER_IP=$(hostname -I | awk '{print $1}')
sed -i "s/<YOUR_IP>/$SERVER_IP/g" config.json
sed -i 's/<YOUR_HOST>/your-domain.com/g' config.json

# Validate JSON
echo "✅ Validating config.json..."
if ! cat config.json | jq '.' > /dev/null 2>&1; then
    echo "❌ Invalid JSON! Restoring backup..."
    cp config.json.backup config.json
    exit 1
fi

cat config.json | jq '.'

# Build
echo "🔨 Building..."
/usr/local/go/bin/go build -o smartsni main.go

if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    exit 1
fi

echo "✅ Build successful!"

# Restart service
echo "🔄 Restarting service..."
systemctl restart sni.service

# Wait
sleep 2

# Check status
echo "📊 Service status:"
systemctl status sni.service --no-pager -l

echo ""
echo "🔍 Checking ports..."
ss -tulnp | grep -E '8080|8088'

echo ""
echo "📝 Recent logs:"
journalctl -u sni.service -n 10 --no-pager

echo ""
echo "✅ Update complete!"
echo "🌐 Web Panel: http://89.47.113.135:8088/panel"
