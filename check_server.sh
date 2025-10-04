#!/bin/bash

# این اسکریپت را روی سرور اجرا کنید تا مشکل را ببینیم

echo "=== بررسی وضعیت سرویس ==="
systemctl status sni.service --no-pager -l

echo ""
echo "=== 50 خط آخر لاگ ==="
journalctl -u sni.service -n 50 --no-pager

echo ""
echo "=== محتوای config.json ==="
cat /root/smartSNI/config.json

echo ""
echo "=== بررسی فایل‌های موجود ==="
ls -la /root/smartSNI/

echo ""
echo "=== بررسی پورت‌ها ==="
ss -tulnp | grep -E '8088|8080|443|853'

echo ""
echo "=== بررسی آخرین git commit ==="
cd /root/smartSNI && git log -1 --oneline

echo ""
echo "=== تست build دستی ==="
cd /root/smartSNI
/usr/local/go/bin/go build -o smartsni-test main.go 2>&1 | head -20
