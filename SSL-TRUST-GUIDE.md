# راهنمای Trust کردن Self-Signed SSL Certificate

## چرا پیام "Your connection is not private" می‌بینم؟

این پیام **طبیعی** است! چون:
- Certificate ما **Self-Signed** است (خودمان امضا کردیم)
- مرورگر آن را نمی‌شناسد (چون از CA معتبر نیست)

**اما**: ارتباط شما **کاملاً امن و رمزنگاری شده (4096-bit RSA)** است! ✅

---

## ⚡ راه حل سریع: قبول کردن موقت

### Chrome / Edge / Brave:
1. روی صفحه خطا، کلیک کنید روی **Advanced** (پیشرفته)
2. کلیک کنید روی **Proceed to dns.dnsoverhttps.site (unsafe)**
3. ✅ تمام! صفحه باز می‌شود

### Firefox:
1. کلیک روی **Advanced** (پیشرفته)
2. کلیک روی **Accept the Risk and Continue**
3. ✅ تمام!

### Safari:
1. کلیک روی **Show Details**
2. کلیک روی **visit this website**
3. تایید کنید
4. ✅ تمام!

---

## 🔒 راه حل دائمی: Trust کردن Certificate

اگر می‌خواهید warning را اصلاً نبینید، certificate را به سیستم اضافه کنید:

### روش 1: دانلود Certificate از سرور

```bash
# SSH به سرور
ssh -p 2112 root@89.47.113.135

# نمایش certificate
cat /etc/ssl/smartsni/fullchain.pem

# کپی کردن محتوا (شروع از -----BEGIN تا -----END)
```

### روش 2: دانلود از مرورگر

**Chrome/Edge:**
1. در صفحه HTTPS، کلیک کنید روی قفل (🔒) یا "Not Secure" در address bar
2. **Certificate** → **Details**
3. **Export** → ذخیره با نام `smartsni.crt`

**Firefox:**
1. کلیک روی قفل → **Connection not secure** → **More information**
2. **View Certificate** → **Download** → **PEM (cert)**

---

## 📥 نصب Certificate در سیستم عامل

### Windows:

1. دابل کلیک روی فایل `.crt` یا `.pem`
2. **Install Certificate**
3. **Current User** → Next
4. **Place all certificates in the following store**
5. **Browse** → انتخاب **Trusted Root Certification Authorities**
6. Next → Finish
7. تایید Security Warning
8. ✅ ریستارت مرورگر

### macOS:

```bash
# اضافه کردن به Keychain
sudo security add-trusted-cert -d -r trustRoot \
  -k /Library/Keychains/System.keychain smartsni.crt
```

یا:
1. دابل کلیک روی فایل `.pem`
2. Keychain Access باز می‌شود
3. پیدا کردن "*.dnsoverhttps.site"
4. دابل کلیک → **Trust** → **Always Trust**
5. ✅ ریستارت مرورگر

### Linux (Ubuntu/Debian):

```bash
# کپی certificate
sudo cp smartsni.crt /usr/local/share/ca-certificates/smartsni.crt

# به‌روزرسانی CA store
sudo update-ca-certificates

# برای Firefox (جداگانه)
# Firefox → Settings → Privacy & Security → Certificates → View Certificates
# → Authorities → Import → انتخاب فایل
```

---

## 🌐 Trust در مرورگرها

### Chrome/Edge/Brave:
بعد از نصب در سیستم عامل، به صورت خودکار trust می‌شود.

### Firefox:
باید جداگانه import کنید:
1. Settings → Privacy & Security
2. **Certificates** → **View Certificates**
3. **Authorities** tab → **Import**
4. انتخاب فایل certificate
5. ✅ تیک بزنید: "Trust this CA to identify websites"

---

## 🔐 بررسی امنیت اتصال

بعد از trust کردن، می‌توانید بررسی کنید:

```bash
# تست SSL connection
openssl s_client -connect dns.dnsoverhttps.site:443 -servername dns.dnsoverhttps.site

# خروجی باید نشان دهد:
# - Protocol: TLSv1.3 (یا TLSv1.2)
# - Cipher: ECDHE-RSA-AES256-GCM-SHA384 (یا مشابه)
# - Verify return code: 0 (ok) یا 18 (self signed)
```

---

## ⚠️ هشدارهای مهم

1. **Self-Signed Certificate برای Production**:
   - برای استفاده شخصی: ✅ کاملاً مناسب
   - برای استفاده عمومی: ❌ توصیه نمی‌شود

2. **امنیت**:
   - رمزنگاری 4096-bit RSA قوی است ✅
   - ارتباط کاملاً امن است ✅
   - فقط مرورگر شناسایی نمی‌کند (چون self-signed است)

3. **اعتبار**:
   - این certificate تا 2035 اعتبار دارد
   - نیازی به تمدید ندارد

---

## 🎯 راه حل نهایی: Let's Encrypt (اگر ممکن باشد)

برای اینکه مرورگرها به صورت خودکار trust کنند، نیاز به certificate از یک CA معتبر مثل Let's Encrypt دارید.

**مشکل فعلی**: فایروال خارجی سرور پورت 80 را block کرده

**راه حل‌ها**:
1. ✅ باز کردن پورت 80 در فایروال سرور میزبان
2. ✅ استفاده از DNS Challenge (نیاز به API دسترسی به DNS provider)
3. ✅ استفاده از CloudFlare Tunnel

اگر یکی از اینها امکان‌پذیر شود، می‌توانیم Let's Encrypt SSL دریافت کنیم.

---

## 📞 سوالات متداول

**Q: آیا اتصال من امن است؟**
A: بله! کاملاً امن و رمزنگاری شده است (TLS 1.2/1.3 با 4096-bit RSA)

**Q: چرا مرورگر warning نشان می‌دهد؟**
A: فقط به این دلیل که certificate را نمی‌شناسد. امنیت رمزنگاری تاثیری نمی‌خورد.

**Q: آیا باید certificate را trust کنم؟**
A: اگر این سرور متعلق به شماست، بله! کاملاً امن است.

**Q: هر بار باید "Proceed anyway" بزنم؟**
A: اگر certificate را trust کنید (روش بالا)، دیگر warning نمی‌بینید.

---

## ✅ خلاصه

| روش | سادگی | دائمی | توصیه |
|-----|--------|-------|-------|
| Proceed anyway | ⭐⭐⭐ | ❌ | برای تست |
| Trust در مرورگر | ⭐⭐ | ✅ | برای استفاده شخصی |
| Trust در سیستم | ⭐ | ✅ | برای استفاده دائمی |
| Let's Encrypt | ⭐ | ✅ | بهترین (اگر ممکن باشد) |

---

**🎉 سرور شما با SSL کاملاً امن راه‌اندازی شده است!**
