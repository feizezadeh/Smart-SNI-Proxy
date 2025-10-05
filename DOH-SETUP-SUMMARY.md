# خلاصه راه‌اندازی DoH
# DoH Setup Summary

## ✅ وضعیت فعلی / Current Status

### سرویس‌های در حال اجرا / Running Services

1. **SmartSNI DoH Server** ✅
   - پورت: `localhost:8080`
   - پروتکل: RFC 8484 (DNS wire format)
   - وضعیت: Active

2. **Nginx Reverse Proxy** ✅
   - پورت: `8443` (HTTPS)
   - دامنه: `doh.dnsoverhttps.site`
   - Proxy به: `http://127.0.0.1:8080`
   - SSL: Let's Encrypt

3. **SSL Certificate** ✅
   - صادر شده برای: `doh.dnsoverhttps.site`
   - مسیر: `/etc/letsencrypt/live/doh.dnsoverhttps.site/`

## 🔗 آدرس نهایی DoH / Final DoH URL

```
https://doh.dnsoverhttps.site:8443/dns-query
```

## 📋 فایل‌های پیکربندی / Configuration Files

### 1. Nginx Config
**مسیر**: `/etc/nginx/sites-available/smartsni-doh`

```nginx
upstream dohloop {
    zone dohloop 64k;
    server 127.0.0.1:8080;
    keepalive 32;
}

server {
    listen 8443 ssl http2;
    server_name doh.dnsoverhttps.site;

    ssl_certificate /etc/letsencrypt/live/doh.dnsoverhttps.site/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/doh.dnsoverhttps.site/privkey.pem;
    include /etc/letsencrypt/options-ssl-nginx.conf;
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;

    # Security headers
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "no-referrer" always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload" always;

    # DoH endpoint
    location /dns-query {
        proxy_pass http://dohloop;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        proxy_connect_timeout 10s;
        proxy_send_timeout 10s;
        proxy_read_timeout 10s;

        proxy_buffering off;
        proxy_request_buffering off;
    }

    # Health check
    location /health {
        proxy_pass http://dohloop;
        proxy_http_version 1.1;
        access_log off;
    }
}
```

**فعال‌سازی**:
```bash
ln -sf /etc/nginx/sites-available/smartsni-doh /etc/nginx/sites-enabled/
nginx -t
systemctl reload nginx
```

### 2. SmartSNI Config
**مسیر**: `~/smartSNI/config.json`

```json
{
  "host": "dns.dnsoverhttps.site",
  "domains": {
    "*.youtube.com": "<YOUR_IP>",
    "*.google.com": "<YOUR_IP>"
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
  "user_management": false
}
```

## ⚠️ مشکل فعلی / Current Issue

### Rate Limiting (محدودیت نرخ)
لاگ‌ها نشان می‌دهند:
```
"DoH per-IP rate limit exceeded"
```

**راه حل**: افزایش rate limit در `config.json`:
```json
"rate_limit_per_ip": 100,      // بجای 10
"rate_limit_burst_ip": 200     // بجای 20
```

سپس restart سرویس:
```bash
systemctl restart sni
```

## 🔧 دستورات مفید / Useful Commands

### بررسی وضعیت / Check Status
```bash
# DoH Service
systemctl status sni
journalctl -u sni -f

# Nginx
systemctl status nginx
nginx -t

# Ports
ss -tlnp | grep 8443
ss -tlnp | grep 8080
```

### تست DoH / Test DoH
```bash
# Health check
curl -I https://doh.dnsoverhttps.site:8443/health

# DoH test (نیاز به DNS wire format)
# از مرورگر یا DoH client استفاده کنید
```

### لاگ‌ها / Logs
```bash
# DoH logs
journalctl -u sni -n 50 --no-pager | grep DoH

# Nginx logs
tail -f /var/log/nginx/error.log
tail -f /var/log/nginx/access.log
```

### Restart Services
```bash
systemctl restart sni
systemctl reload nginx
```

## 📝 مراحل نصب (برای مستندسازی) / Installation Steps

### 1. ایجاد Subdomain
```bash
# در DNS provider خود:
# A record: doh.dnsoverhttps.site → 89.47.113.135
```

### 2. دریافت SSL Certificate
```bash
systemctl stop nginx
certbot certonly --standalone -d doh.dnsoverhttps.site
systemctl start nginx
```

### 3. پیکربندی Nginx
```bash
# ایجاد فایل config (محتوای بالا)
nano /etc/nginx/sites-available/smartsni-doh

# فعال‌سازی
ln -sf /etc/nginx/sites-available/smartsni-doh /etc/nginx/sites-enabled/

# تست و reload
nginx -t
systemctl reload nginx
```

### 4. باز کردن پورت در فایروال
```bash
ufw allow 8443/tcp
ufw status
```

### 5. تست نهایی
```bash
curl -I https://doh.dnsoverhttps.site:8443/health
```

## 🎯 مراحل باقی‌مانده / Remaining Steps

1. ✅ Nginx config ایجاد شد
2. ✅ SSL certificate دریافت شد
3. ✅ DoH در حال اجرا است
4. ⚠️ Rate limit باید افزایش یابد
5. ⏳ تست با DoH client (Firefox/Chrome)
6. ⏳ فعال‌سازی user management (اختیاری)

## 📚 مستندات اضافی / Additional Documentation

- `DOH-USAGE.md` - راهنمای کامل استفاده از DoH
- `test-doh.sh` - اسکریپت تست DoH
- `nginx.conf` - پیکربندی اصلی nginx
- `config.json` - پیکربندی SmartSNI

## 🔒 نکات امنیتی / Security Notes

1. **HTTPS Only**: همه ارتباطات روی HTTPS
2. **Rate Limiting**: محدودیت نرخ فعال (باید تنظیم شود)
3. **User Management**: می‌تواند فعال شود برای کنترل دسترسی
4. **Security Headers**: تمام headerهای امنیتی فعال
5. **TLS 1.2/1.3**: فقط پروتکل‌های امن

---

**تاریخ آخرین بروزرسانی**: 2025-10-05
**وضعیت**: DoH آماده استفاده (نیاز به تنظیم rate limit)
