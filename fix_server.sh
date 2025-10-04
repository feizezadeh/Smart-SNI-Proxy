#!/bin/bash

echo "🔧 در حال رفع مشکلات..."

cd /root/smartSNI

# 1. مطمئن شویم که در main branch هستیم
echo "1️⃣ بررسی branch..."
git checkout main
git fetch origin
git reset --hard origin/main

# 2. پاک کردن باینری قدیمی
echo "2️⃣ پاک کردن باینری قدیمی..."
rm -f smartsni

# 3. Fix کردن config.json
echo "3️⃣ Fix کردن config.json..."
SERVER_IP=$(hostname -I | awk '{print $1}')

# ساخت config جدید با مقادیر صحیح
cat > config.json << EOF
{
  "host": "dns.dnsoverhttps.site",
  "domains": {
    "youtube.com": "$SERVER_IP",
    "*.youtube.com": "$SERVER_IP",
    "google.com": "$SERVER_IP",
    "*.google.com": "$SERVER_IP"
  },
  "upstream_doh": [
    "https://1.1.1.1/dns-query",
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
  "web_panel_password": "8c6976e5b5410415bde908bd4dee15dfb167a9c873fc4bb8a81f6f2ab448a918",
  "web_panel_port": 8088,
  "user_management": false
}
EOF

echo "✅ Config ساخته شد:"
cat config.json | jq '.'

# 4. Build
echo ""
echo "4️⃣ Building..."
/usr/local/go/bin/go build -o smartsni main.go

if [ $? -ne 0 ]; then
    echo "❌ Build failed! خطاها:"
    /usr/local/go/bin/go build -o smartsni main.go
    exit 1
fi

echo "✅ Build موفق"

# 5. بررسی باینری
echo ""
echo "5️⃣ بررسی باینری..."
ls -lh smartsni
file smartsni

# 6. Stop کردن سرویس قدیمی
echo ""
echo "6️⃣ Stopping old service..."
systemctl stop sni.service
sleep 1

# 7. Start کردن سرویس
echo "7️⃣ Starting service..."
systemctl start sni.service
sleep 2

# 8. بررسی وضعیت
echo ""
echo "8️⃣ وضعیت سرویس:"
systemctl status sni.service --no-pager -l | head -20

# 9. بررسی پورت‌ها
echo ""
echo "9️⃣ بررسی پورت‌ها:"
ss -tulnp | grep -E '8088|8080'

# 10. نمایش لاگ‌های اخیر
echo ""
echo "🔟 لاگ‌های اخیر:"
journalctl -u sni.service -n 15 --no-pager

echo ""
echo "✅ تمام!"
echo ""
echo "اگر service فعال است:"
echo "🌐 Web Panel: http://$SERVER_IP:8088/panel"
echo "👤 Username: admin"
echo "🔑 Password: admin"
