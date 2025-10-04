# Manual Deployment Guide

## چگونه فایل‌ها را به سرور منتقل کنیم

### روش 1: Git Pull (توصیه می‌شود)

```bash
# SSH به سرور
ssh root@89.47.113.135

# رفتن به دایرکتوری پروژه
cd /root/smartSNI

# دریافت آخرین تغییرات
git pull origin main

# Build مجدد
/usr/local/go/bin/go build -o smartsni main.go

# اگر config مشکل دارد، fix کنید
sed -i 's/<YOUR_IP>/89.47.113.135/g' config.json
sed -i 's/<YOUR_HOST>/your-domain.com/g' config.json

# Restart سرویس
systemctl restart sni.service

# بررسی وضعیت
systemctl status sni.service
```

### روش 2: Manual Upload

اگر git کار نکرد، فایل‌ها را دستی آپلود کنید:

```bash
# از کامپیوتر محلی (Linux/Mac)
scp /home/mehdi/DOH/smartSNI/main.go root@89.47.113.135:/root/smartSNI/
scp /home/mehdi/DOH/smartSNI/webpanel.html root@89.47.113.135:/root/smartSNI/

# سپس SSH به سرور و rebuild
ssh root@89.47.113.135
cd /root/smartSNI
/usr/local/go/bin/go build -o smartsni main.go
systemctl restart sni.service
```

## تست تغییرات

### 1. بررسی Web Panel

```bash
# روی سرور
curl http://localhost:8088/panel
```

باید HTML برگرداند (نه 404 یا خطا).

### 2. تست Add Domain API

```bash
# دریافت session ID از browser (F12 > Application > LocalStorage)
SESSION_ID="your-session-id-here"

# تست API
curl -X POST http://localhost:8088/panel/api/domains/add \
  -H "X-Session-ID: $SESSION_ID" \
  -H "Content-Type: application/json" \
  -d '{"domain":"test.com"}'
```

باید پاسخ: `{"status":"added"}` برگردد.

### 3. تست User Management API

```bash
curl -X GET http://localhost:8088/panel/api/users \
  -H "X-Session-ID: $SESSION_ID"
```

باید JSON با لیست کاربران برگردد (حتی اگر خالی باشد).

## رفع مشکلات رایج

### خطا: "Failed to load config"

```bash
cd /root/smartSNI

# بررسی صحت JSON
cat config.json | jq '.'

# اگر خطا داشت، مقادیر <YOUR_IP> را جایگزین کنید
SERVER_IP=$(hostname -I | awk '{print $1}')
sed -i "s/<YOUR_IP>/$SERVER_IP/g" config.json

# بررسی مجدد
cat config.json | jq '.'

# Restart
systemctl restart sni.service
```

### خطا: "Connection error"

این خطا معمولاً به دلایل زیر است:

1. **Session نامعتبر**: از browser خارج و دوباره login کنید
2. **CORS issue**: مطمئن شوید از همان IP/domain دسترسی دارید
3. **API endpoint اشتباه**: مسیر API را چک کنید

**تست:**
```bash
# بررسی logs
journalctl -u sni.service -f

# در یک terminal دیگر، از browser عملیات را انجام دهید
# باید logs را ببینید
```

### خطا: Service کرش می‌کند

```bash
# بررسی logs
journalctl -u sni.service -n 50

# معمولاً یکی از این مشکلات است:
# 1. Invalid IP in config
# 2. Invalid JSON
# 3. Port already in use

# Check ports
ss -tulnp | grep -E '8080|8088|443|853'

# اگر port در استفاده است:
lsof -i :8088
# سپس process را kill کنید
```

## تغییرات اخیر (فایل‌هایی که باید update شوند)

### main.go
- بهبود error handling در `handlePanelAddDomain`
- اضافه شدن User Management API endpoints
- سیستم invitation token

### webpanel.html
- اضافه شدن بخش User Management
- حذف فیلد IP از add domain form
- اضافه شدن توابع `loadUsers()`, `createInvite()`, etc.

### config.json
- اضافه شدن `user_management: false`

## Checklist نصب موفق

- [ ] فایل‌ها به‌روز شدند (git pull یا manual upload)
- [ ] Build موفقیت‌آمیز بود
- [ ] config.json valid JSON است
- [ ] همه <YOUR_IP> جایگزین شدند
- [ ] سرویس active است: `systemctl status sni.service`
- [ ] Port 8088 listen می‌کند: `ss -tulnp | grep 8088`
- [ ] Web Panel باز می‌شود: http://89.47.113.135:8088/panel
- [ ] Login کار می‌کند (admin/admin)
- [ ] بخش User Management نمایش داده می‌شود
- [ ] می‌توانید domain اضافه کنید (بدون نیاز به IP)

## دستورات سریع

```bash
# همه در یک خط
cd /root/smartSNI && git pull && /usr/local/go/bin/go build -o smartsni main.go && sed -i 's/<YOUR_IP>/89.47.113.135/g' config.json && systemctl restart sni.service && systemctl status sni.service
```

## نکات مهم

1. **قبل از restart**: همیشه `systemctl status sni.service` را چک کنید
2. **بعد از update**: حتماً rebuild کنید (go build)
3. **config.json**: همیشه با `jq` validate کنید
4. **logs**: برای debug از `journalctl -u sni.service -f` استفاده کنید

## تماس برای پشتیبانی

اگر مشکلی حل نشد:

1. لاگ‌های کامل را جمع‌آوری کنید:
```bash
journalctl -u sni.service -n 100 > /root/smartsni-logs.txt
cat /root/smartSNI/config.json > /root/smartsni-config.txt
```

2. بررسی کنید که کدام endpoint مشکل دارد
3. از browser DevTools (F12 > Network) خطاها را ببینید
