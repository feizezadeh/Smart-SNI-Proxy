# راهنمای استفاده از DoH (DNS over HTTPS)
# DoH (DNS over HTTPS) Usage Guide

## 🔧 تنظیمات فعلی / Current Setup

### ✅ آنچه انجام شده / What's Configured

1. **DoH Server**: در حال اجرا روی `localhost:8080`
   - پیاده‌سازی استاندارد RFC 8484
   - پشتیبانی از GET و POST

2. **Nginx Reverse Proxy**: پورت 8443
   - SSL termination با Let's Encrypt
   - Proxy به `localhost:8080`
   - دامنه: `doh.dnsoverhttps.site`

3. **SSL Certificate**:
   - دامنه: `doh.dnsoverhttps.site`
   - صادر شده توسط Let's Encrypt
   - مسیر: `/etc/letsencrypt/live/doh.dnsoverhttps.site/`

### 📍 Endpoint URL

```
https://doh.dnsoverhttps.site:8443/dns-query
```

## 🚀 نحوه استفاده / How to Use

### روش 1: استفاده در مرورگر / Browser Configuration

#### Firefox
1. Settings → Privacy & Security → DNS over HTTPS
2. فعال کردن "Enable DNS over HTTPS"
3. انتخاب "Custom" و وارد کردن URL:
   ```
   https://doh.dnsoverhttps.site:8443/dns-query
   ```

#### Chrome/Edge
1. Settings → Privacy and security → Security
2. فعال کردن "Use secure DNS"
3. انتخاب "Custom" و وارد کردن:
   ```
   https://doh.dnsoverhttps.site:8443/dns-query
   ```

### روش 2: استفاده با curl

#### GET Request (با DNS wire format به صورت base64url):
```bash
# نیاز به تولید DNS query packet دارد
curl -H 'accept: application/dns-message' \
  'https://doh.dnsoverhttps.site:8443/dns-query?dns=<BASE64URL_ENCODED_DNS_QUERY>'
```

#### POST Request (با DNS wire format خام):
```bash
# ارسال DNS packet به صورت binary
curl -X POST -H 'content-type: application/dns-message' \
  --data-binary @query.bin \
  'https://doh.dnsoverhttps.site:8443/dns-query'
```

### روش 3: استفاده با DoH Client Tools

#### cloudflared
```bash
cloudflared proxy-dns \
  --upstream https://doh.dnsoverhttps.site:8443/dns-query
```

#### dnsproxy
```bash
dnsproxy -u https://doh.dnsoverhttps.site:8443/dns-query
```

## ⚙️ تنظیمات / Configuration

### Rate Limiting (محدودیت نرخ درخواست)
```json
"rate_limit_per_ip": 10,        // 10 درخواست در ثانیه
"rate_limit_burst_ip": 20       // حداکثر 20 درخواست burst
```

⚠️ **توجه**: برای استفاده عمومی، این مقادیر را افزایش دهید.

### User Management (مدیریت کاربران)
```json
"user_management": false   // غیرفعال (همه دسترسی دارند)
```

اگر `true` باشد، فقط IP‌های ثبت شده دسترسی خواهند داشت.

## 🔍 بررسی وضعیت / Status Check

### چک کردن Nginx
```bash
systemctl status nginx
ss -tlnp | grep 8443
```

### چک کردن DoH Service
```bash
systemctl status sni
journalctl -u sni -f   # نمایش لاگ‌های زنده
```

### تست اتصال
```bash
# تست پورت 8443
curl -I https://doh.dnsoverhttps.site:8443/dns-query

# تست DoH محلی (بدون SSL)
dig @127.0.0.1 -p 8080 google.com
```

## 📊 لاگ‌ها و Monitoring

### مشاهده لاگ‌های DoH
```bash
journalctl -u sni -n 100 --no-pager | grep DoH
```

### لاگ‌های رایج / Common Logs

✅ **درخواست موفق**:
```
"level":"INFO","msg":"DoH request","client":"x.x.x.x"
```

❌ **Rate Limit**:
```
"level":"WARN","msg":"DoH per-IP rate limit exceeded","client":"x.x.x.x"
```

❌ **احراز هویت ناموفق** (وقتی user_management فعال است):
```
"level":"WARN","msg":"DoH user not authorized","client":"x.x.x.x"
```

## 🔐 امنیت / Security

### Headers امنیتی فعال:
- X-Content-Type-Options: nosniff
- X-Frame-Options: DENY
- X-XSS-Protection: 1; mode=block
- Referrer-Policy: no-referrer
- Strict-Transport-Security (HSTS)

### SSL/TLS:
- پروتکل‌ها: TLSv1.2, TLSv1.3
- HTTP/2 فعال

## 🐛 عیب‌یابی / Troubleshooting

### خطای "Not Found"
- **علت**: فرمت query اشتباه (باید DNS wire format باشد، نه JSON)
- **راه حل**: از DoH client استفاده کنید یا DNS packet درست تولید کنید

### خطای "Rate limit exceeded"
- **علت**: تعداد درخواست بیش از حد مجاز
- **راه حل**: افزایش `rate_limit_per_ip` در config.json

### خطای SSL
- **علت**: گواهی منقضی یا نامعتبر
- **راه حل**: تمدید گواهی با `certbot renew`

### پورت 8443 در دسترس نیست
- **علت**: nginx اجرا نشده یا فایروال
- **راه حل**:
  ```bash
  systemctl start nginx
  ufw allow 8443/tcp
  ```

## 📝 یادداشت‌های مهم / Important Notes

1. **فرمت DoH**: این سرور از RFC 8484 استفاده می‌کند (DNS wire format)، نه Google DNS JSON API

2. **پورت 443**: پورت 443 توسط SNI Proxy استفاده می‌شود، بنابراین DoH روی پورت 8443 است

3. **User Management**: اگر فعال شود، کاربران باید ابتدا IP خود را ثبت کنند:
   - Admin کاربر می‌سازد
   - کاربر از لینک ثبت‌نام استفاده می‌کند
   - IP کاربر ثبت می‌شود (حداکثر تعداد IP با FIFO)

4. **Rate Limiting**: برای استفاده عمومی، حتماً rate limit را افزایش دهید

## 🔗 لینک‌های مفید / Useful Links

- RFC 8484: https://tools.ietf.org/html/rfc8484
- DNS Wire Format: https://www.ietf.org/rfc/rfc1035.txt
- Let's Encrypt: https://letsencrypt.org/
