# Smart SNI Proxy v2.0

<div align="center">

![Version](https://img.shields.io/badge/version-2.0-blue.svg)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

**پروکسی SNI هوشمند با DoH/DoT و مدیریت کاربران**

[فارسی](#فارسی) | [English](#english)

</div>

---

## فارسی

### 📋 فهرست مطالب
- [ویژگی‌ها](#ویژگیها)
- [نصب سریع](#نصب-سریع)
- [پیکربندی](#پیکربندی)
- [مدیریت کاربران](#مدیریت-کاربران)
- [راه‌اندازی DoH](#راهاندازی-doh)
- [مستندات](#مستندات)

### ✨ ویژگی‌ها

#### 🔐 پروکسی SNI (Server Name Indication)
- پروکسی شفاف TLS برای دامنه‌های مشخص
- پشتیبانی از wildcard domains
- مسیریابی بر اساس SNI بدون رمزگشایی TLS

#### 🌐 DNS over HTTPS (DoH)
- پیاده‌سازی RFC 8484 استاندارد
- پشتیبانی از GET و POST
- نصب اتوماتیک SSL با Let's Encrypt
- nginx reverse proxy برای عملکرد بهتر

#### 🔒 DNS over TLS (DoT)
- سرور DoT روی پورت 853
- رمزگذاری کامل ترافیک DNS
- پشتیبانی از چندین upstream DNS

#### 👥 مدیریت کاربران (User Management)
- ایجاد کاربر با تاریخ انقضا
- محدودیت تعداد IP برای هر کاربر
- **FIFO (First In First Out)**: حذف خودکار قدیمی‌ترین IP هنگام پر شدن
- لینک ثبت‌نام یکتا برای هر کاربر
- کنترل دسترسی بر اساس IP
- Web Panel برای مدیریت آسان

#### 📊 Web Panel
- رابط کاربری ساده و زیبا
- ایجاد و مدیریت کاربران
- نمایش آمار استفاده
- لیست IP‌های ثبت شده
- فعال/غیرفعال کردن کاربران

#### ⚡ Performance & Security
- Cache DNS با TTL قابل تنظیم
- Rate limiting (per-IP و global)
- Metrics و monitoring
- Security headers کامل
- Connection pooling و keepalive

### 🚀 نصب سریع

#### پیش‌نیازها
- سیستم عامل: Ubuntu/Debian/CentOS/Fedora
- دسترسی root
- دامنه با رکورد A تنظیم شده
- پورت‌های باز: 443, 853, 8080, 8088, 8443

#### نصب با یک دستور:
```bash
bash <(curl -s https://raw.githubusercontent.com/feizezadeh/Smart-SNI-Proxy/main/install.sh)
```

#### مراحل نصب:
1. انتخاب گزینه `1` (Install)
2. وارد کردن دامنه اصلی
3. وارد کردن وب‌سایت‌های مورد نظر (مثل: `youtube.com,google.com`)
4. انتخاب راه‌اندازی DoH subdomain (پیشنهاد: بله)
5. وارد کردن subdomain برای DoH (مثل: `doh.example.com`)
6. صبر برای اتمام نصب

#### خروجی نصب:
```
Smart SNI v2.0 Installed Successfully!
_______________________________________

📡 Endpoints:
  DoH:  https://example.com/dns-query
  DoH (dedicated): https://doh.example.com:8443/dns-query
  DoT:  example.com:853
  SNI:  example.com:443

🌐 Web Panel:
  URL:      http://SERVER_IP:8088
  Username: admin
  Password: [رمز عبور تصادفی]
  ⚠️  Save this password! It's only shown once.

👥 User Management:
  Status: Disabled (user_management: false)
  To enable: Set 'user_management: true' in config.json
  Features: IP-based access control with FIFO
```

### ⚙️ پیکربندی

#### فایل config.json:
```json
{
  "host": "example.com",
  "domains": {
    "youtube.com": "SERVER_IP",
    "*.youtube.com": "SERVER_IP",
    "google.com": "SERVER_IP",
    "*.google.com": "SERVER_IP"
  },
  "upstream_doh": [
    "https://1.1.1.1/dns-query",
    "https://8.8.8.8/dns-query"
  ],
  "enable_auth": false,
  "auth_tokens": [],
  "cache_ttl": 300,
  "rate_limit_per_ip": 100,
  "rate_limit_burst_ip": 200,
  "log_level": "info",
  "web_panel_enabled": true,
  "web_panel_username": "admin",
  "web_panel_password": "SHA256_HASH",
  "web_panel_port": 8088,
  "user_management": false
}
```

#### تنظیمات مهم:
- `rate_limit_per_ip`: تعداد درخواست در ثانیه (پیش‌فرض: 100)
- `user_management`: فعال/غیرفعال کردن کنترل دسترسی کاربران
- `cache_ttl`: مدت زمان cache DNS (ثانیه)

### 👥 مدیریت کاربران

#### فعال‌سازی:
1. اجرای اسکریپت نصب
2. انتخاب گزینه `6` (User Management)
3. انتخاب گزینه `1` (Enable User Management)

#### ایجاد کاربر:
1. باز کردن Web Panel: `http://SERVER_IP:8088`
2. ورود با username: `admin` و رمز عبور نصب
3. کلیک روی "Create User"
4. وارد کردن:
   - نام کاربر
   - توضیحات (اختیاری)
   - حداکثر تعداد IP مجاز
   - تعداد روزهای اعتبار
5. کپی کردن لینک ثبت‌نام و ارسال به کاربر

#### ثبت IP توسط کاربر:
کاربر با باز کردن لینک ثبت‌نام، IP خود را ثبت می‌کند:
```
http://SERVER_IP:8088/register?token=USER_ID
```

#### سیستم FIFO:
- هر کاربر حداکثر تعداد IP مشخصی دارد
- هنگام ثبت IP جدید و پر بودن ظرفیت:
  - قدیمی‌ترین IP حذف می‌شود
  - IP جدید اضافه می‌شود
- مثال: max_ips=3 → [IP1, IP2, IP3] + IP4 → [IP2, IP3, IP4]

### 🌐 راه‌اندازی DoH

#### روش 1: حین نصب (پیشنهادی)
اسکریپت نصب به صورت خودکار:
- Subdomain ایجاد می‌کند
- SSL certificate دریافت می‌کند
- nginx config می‌سازد
- پورت 8443 را باز می‌کند

#### روش 2: دستی
مستندات کامل در: [DOH-SETUP-SUMMARY.md](DOH-SETUP-SUMMARY.md)

#### استفاده از DoH:

**در مرورگر Firefox:**
1. Settings → Privacy & Security
2. DNS over HTTPS → Enable
3. Custom → `https://doh.example.com:8443/dns-query`

**در Chrome/Edge:**
1. Settings → Privacy and security
2. Use secure DNS → Custom
3. `https://doh.example.com:8443/dns-query`

**با curl (نیاز به DNS wire format):**
```bash
curl -H 'accept: application/dns-message' \
  'https://doh.example.com:8443/dns-query?dns=BASE64_ENCODED_DNS_QUERY'
```

### 📊 مانیتورینگ

#### مشاهده لاگ‌ها:
```bash
# Live logs
journalctl -u sni.service -f

# آخرین 50 لاگ
journalctl -u sni.service -n 50

# فقط خطاها
journalctl -u sni.service -p err
```

#### Metrics:
```bash
# Health check
curl http://127.0.0.1:8080/health

# Metrics
curl http://127.0.0.1:8080/metrics | jq
```

#### وضعیت سرویس:
```bash
systemctl status sni.service
```

### 🛠️ دستورات مفید

#### مدیریت سرویس:
```bash
# شروع
systemctl start sni.service

# توقف
systemctl stop sni.service

# ریستارت
systemctl restart sni.service

# Reload (بدون قطع اتصال)
curl -X POST http://127.0.0.1:8080/admin/reload
```

#### اضافه کردن دامنه جدید:
```bash
# اجرای اسکریپت نصب
bash /root/smartSNI/install.sh

# انتخاب گزینه 4 (Add Sites)
```

#### بررسی پورت‌ها:
```bash
# DoH/DoT
ss -tlnp | grep -E '8080|853'

# Web Panel
ss -tlnp | grep 8088

# DoH SSL
ss -tlnp | grep 8443

# SNI
ss -tlnp | grep 443
```

### 📚 مستندات

- **[DOH-USAGE.md](DOH-USAGE.md)** - راهنمای جامع استفاده از DoH
- **[DOH-SETUP-SUMMARY.md](DOH-SETUP-SUMMARY.md)** - خلاصه راه‌اندازی DoH
- **[test-doh.sh](test-doh.sh)** - اسکریپت تست DoH

### 🔧 عیب‌یابی

#### مشکلات رایج:

**1. سرویس شروع نمی‌شود:**
```bash
# بررسی لاگ‌ها
journalctl -u sni.service -n 50

# بررسی پیکربندی
jq '.' /root/smartSNI/config.json

# بررسی باینری
ls -lh /root/smartSNI/smartsni
```

**2. DoH کار نمی‌کند:**
```bash
# بررسی nginx
nginx -t
systemctl status nginx

# بررسی پورت 8443
ss -tlnp | grep 8443

# بررسی SSL
curl -I https://doh.example.com:8443/dns-query
```

**3. Rate limit:**
```bash
# افزایش rate limit در config.json
jq '.rate_limit_per_ip = 200' config.json > temp.json
mv temp.json config.json
systemctl restart sni.service
```

**4. Web Panel دسترسی ندارد:**
```bash
# بررسی فایروال
ufw status
ufw allow 8088/tcp

# بررسی سرویس
ss -tlnp | grep 8088
curl http://127.0.0.1:8088
```

### 🔐 امنیت

#### توصیه‌های امنیتی:
1. **رمز عبور قوی**: رمز عبور Web Panel را تغییر دهید
2. **فایروال**: فقط پورت‌های لازم را باز کنید
3. **SSL**: حتماً از HTTPS استفاده کنید
4. **Rate Limiting**: مقادیر مناسب تنظیم کنید
5. **User Management**: برای کنترل دسترسی فعال کنید
6. **به‌روزرسانی**: به طور منظم سیستم را به‌روز کنید

#### Security Headers فعال:
- X-Content-Type-Options: nosniff
- X-Frame-Options: DENY
- X-XSS-Protection: 1; mode=block
- Referrer-Policy: no-referrer
- Strict-Transport-Security (HSTS)

### 🤝 مشارکت

Pull Request ها و Issue ها خوش آمدید! لطفاً:
1. Fork کنید
2. تغییرات خود را اعمال کنید
3. Pull Request ارسال کنید

### 📜 لایسنس

MIT License - برای جزئیات بیشتر فایل LICENSE را ببینید

### 🙏 تشکر

این پروژه از پروژه‌های زیر الهام گرفته شده:
- [Original Smart SNI](https://github.com/Ptechgithub/smartSNI)
- Cloudflare DNS
- RFC 8484 (DoH Standard)

---

## English

### 📋 Table of Contents
- [Features](#features)
- [Quick Install](#quick-install)
- [Configuration](#configuration)
- [User Management](#user-management-1)
- [DoH Setup](#doh-setup)
- [Documentation](#documentation-1)

### ✨ Features

#### 🔐 SNI Proxy (Server Name Indication)
- Transparent TLS proxy for specified domains
- Wildcard domain support
- SNI-based routing without TLS decryption

#### 🌐 DNS over HTTPS (DoH)
- RFC 8484 standard implementation
- GET and POST support
- Auto SSL setup with Let's Encrypt
- nginx reverse proxy for better performance

#### 🔒 DNS over TLS (DoT)
- DoT server on port 853
- Full DNS traffic encryption
- Multiple upstream DNS support

#### 👥 User Management
- Create users with expiration dates
- IP limit per user
- **FIFO (First In First Out)**: Auto-remove oldest IP when full
- Unique registration link per user
- IP-based access control
- Web Panel for easy management

#### 📊 Web Panel
- Simple and beautiful UI
- Create and manage users
- View usage statistics
- List registered IPs
- Enable/disable users

#### ⚡ Performance & Security
- DNS cache with configurable TTL
- Rate limiting (per-IP and global)
- Metrics and monitoring
- Complete security headers
- Connection pooling and keepalive

### 🚀 Quick Install

#### Prerequisites
- OS: Ubuntu/Debian/CentOS/Fedora
- Root access
- Domain with A record configured
- Open ports: 443, 853, 8080, 8088, 8443

#### One-line installation:
```bash
bash <(curl -s https://raw.githubusercontent.com/feizezadeh/Smart-SNI-Proxy/main/install.sh)
```

#### Installation steps:
1. Select option `1` (Install)
2. Enter main domain
3. Enter target websites (e.g., `youtube.com,google.com`)
4. Setup DoH subdomain (recommended: yes)
5. Enter subdomain for DoH (e.g., `doh.example.com`)
6. Wait for installation to complete

### ⚙️ Configuration

See Persian section above for config.json details.

### 👥 User Management

#### Enable:
1. Run install script
2. Select option `6` (User Management)
3. Select option `1` (Enable User Management)

#### Create User:
1. Open Web Panel: `http://SERVER_IP:8088`
2. Login with username: `admin` and installation password
3. Click "Create User"
4. Enter:
   - User name
   - Description (optional)
   - Max IPs allowed
   - Valid days
5. Copy registration link and send to user

#### FIFO System:
- Each user has a max IP limit
- When adding new IP and capacity is full:
  - Oldest IP is removed
  - New IP is added
- Example: max_ips=3 → [IP1, IP2, IP3] + IP4 → [IP2, IP3, IP4]

### 🌐 DoH Setup

#### Method 1: During Installation (Recommended)
The install script automatically:
- Creates subdomain
- Obtains SSL certificate
- Creates nginx config
- Opens port 8443

#### Method 2: Manual
Full documentation at: [DOH-SETUP-SUMMARY.md](DOH-SETUP-SUMMARY.md)

#### Using DoH:

**Firefox:**
1. Settings → Privacy & Security
2. DNS over HTTPS → Enable
3. Custom → `https://doh.example.com:8443/dns-query`

**Chrome/Edge:**
1. Settings → Privacy and security
2. Use secure DNS → Custom
3. `https://doh.example.com:8443/dns-query`

### 📚 Documentation

- **[DOH-USAGE.md](DOH-USAGE.md)** - Comprehensive DoH usage guide
- **[DOH-SETUP-SUMMARY.md](DOH-SETUP-SUMMARY.md)** - DoH setup summary
- **[test-doh.sh](test-doh.sh)** - DoH testing script

### 🔐 Security

#### Security Recommendations:
1. **Strong Password**: Change Web Panel password
2. **Firewall**: Only open necessary ports
3. **SSL**: Always use HTTPS
4. **Rate Limiting**: Configure appropriate values
5. **User Management**: Enable for access control
6. **Updates**: Regularly update the system

### 🤝 Contributing

Pull Requests and Issues are welcome! Please:
1. Fork the repository
2. Make your changes
3. Submit a Pull Request

### 📜 License

MIT License - See LICENSE file for details

### 🙏 Credits

This project is inspired by:
- [Original Smart SNI](https://github.com/Ptechgithub/smartSNI)
- Cloudflare DNS
- RFC 8484 (DoH Standard)

---

<div align="center">

**⭐ If you find this project useful, please star it! ⭐**

Made with ❤️ by the community

</div>
