# راه‌اندازی DoH با SSL

## مشکل فعلی

- **SNI Proxy** روی پورت 443 است (برای proxy کردن youtube.com, google.com و...)
- **DoH** روی پورت 8080 (localhost) است و از بیرون در دسترس نیست
- برای دسترسی به DoH با SSL، نیاز به nginx روی پورت 443 داریم

## راه‌حل‌ها

### راه 1: استفاده از Subdomain (پیشنهادی ✅)

استفاده از subdomain جداگانه برای DoH:

```
doh.dnsoverhttps.site  → nginx:443 → localhost:8080 (DoH)
dns.dnsoverhttps.site  → SNI Proxy:443 (برای youtube, google, ...)
```

**مراحل:**

1. یک رکورد DNS اضافه کنید:
   ```
   Type: A
   Name: doh
   Value: 89.47.113.135
   TTL: 3600
   ```

2. اسکریپت را اجرا کنید:
   ```bash
   cd ~/smartSNI
   git pull origin main
   bash setup-doh-ssl.sh
   # و انتخاب کنید: subdomain
   ```

3. استفاده:
   ```
   DoH URL: https://doh.dnsoverhttps.site/dns-query
   ```

### راه 2: استفاده از پورت دیگر برای DoH

DoH را روی پورت 8443 با SSL قرار دهید:

```
https://dns.dnsoverhttps.site:8443/dns-query
```

**مزیت:** نیاز به subdomain ندارد
**معایب:** برخی فایروال‌ها ممکن است پورت 8443 را ببندند

### راه 3: غیرفعال کردن SNI Proxy

اگر به SNI Proxy نیاز ندارید، آن را غیرفعال کنید و nginx را روی 443 قرار دهید.

**نکته:** با این کار youtube.com و google.com دیگر proxy نمی‌شوند.

## توصیه

**راه 1 (Subdomain)** را پیشنهاد می‌کنم:
- ✅ مشکلی با SNI Proxy ندارد
- ✅ از پورت استاندارد 443 استفاده می‌کند
- ✅ با تمام کلاینت‌ها کار می‌کند
- ✅ جدا کردن DoH از SNI Proxy

## تست DoH

بعد از راه‌اندازی:

```bash
# تست با curl
curl -H 'accept: application/dns-json' \
  'https://doh.dnsoverhttps.site/dns-query?name=google.com&type=A'

# تست با dig
dig @doh.dnsoverhttps.site google.com

# در مرورگر (Firefox)
network.trr.mode = 2
network.trr.uri = https://doh.dnsoverhttps.site/dns-query
```
