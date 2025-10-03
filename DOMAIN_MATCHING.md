# 🌐 Domain Matching Guide

## How Domain Matching Works

Smart SNI Proxy v2.0 supports flexible domain matching with both **exact** and **wildcard** patterns.

### Pattern Types

#### 1. Exact Match
Matches only the exact domain name.

```json
{
  "domains": {
    "example.com": "1.2.3.4"
  }
}
```

**Matches:**
- ✅ `example.com`

**Does NOT match:**
- ❌ `www.example.com`
- ❌ `subdomain.example.com`
- ❌ `api.example.com`

#### 2. Wildcard Match
Matches all subdomains using `*.` prefix.

```json
{
  "domains": {
    "*.example.com": "1.2.3.4"
  }
}
```

**Matches:**
- ✅ `www.example.com`
- ✅ `subdomain.example.com`
- ✅ `api.example.com`

**Does NOT match:**
- ❌ `example.com` (no subdomain)

### ⚠️ Important: Use Both for Full Coverage

To properly route **both** the main domain AND its subdomains, you need **both patterns**:

```json
{
  "domains": {
    "example.com": "1.2.3.4",       // Main domain
    "*.example.com": "1.2.3.4"      // All subdomains
  }
}
```

**This matches:**
- ✅ `example.com`
- ✅ `www.example.com`
- ✅ `api.example.com`
- ✅ `cdn.example.com`
- ✅ Any subdomain

---

## Real-World Examples

### Example 1: YouTube
```json
{
  "domains": {
    "youtube.com": "YOUR_SERVER_IP",
    "*.youtube.com": "YOUR_SERVER_IP",
    "googlevideo.com": "YOUR_SERVER_IP",
    "*.googlevideo.com": "YOUR_SERVER_IP"
  }
}
```

**Why both?**
- `youtube.com` - Main site
- `*.youtube.com` - Subdomains like `www.youtube.com`, `m.youtube.com`
- `googlevideo.com` - Video CDN
- `*.googlevideo.com` - Video CDN subdomains

### Example 2: IP Check Sites
```json
{
  "domains": {
    "whatismyipaddress.com": "YOUR_SERVER_IP",
    "*.whatismyipaddress.com": "YOUR_SERVER_IP"
  }
}
```

**Why both?**
- User visits `whatismyipaddress.com` → Shows server IP ✅
- User visits `www.whatismyipaddress.com` → Shows server IP ✅

### Example 3: Google Services
```json
{
  "domains": {
    "google.com": "YOUR_SERVER_IP",
    "*.google.com": "YOUR_SERVER_IP",
    "googleapis.com": "YOUR_SERVER_IP",
    "*.googleapis.com": "YOUR_SERVER_IP",
    "gstatic.com": "YOUR_SERVER_IP",
    "*.gstatic.com": "YOUR_SERVER_IP"
  }
}
```

---

## Automatic Behavior

### Via Web Panel
When you add a domain via the web panel, **both patterns are added automatically**:

```
Input: example.com
Result:
  ✅ example.com → IP
  ✅ *.example.com → IP
```

### Via install.sh
When you add domains during installation, **both patterns are added automatically**:

```bash
Enter Website names: youtube.com,google.com

Result:
  ✅ youtube.com → IP
  ✅ *.youtube.com → IP
  ✅ google.com → IP
  ✅ *.google.com → IP
```

### Manual Configuration
If you edit `config.json` manually, **remember to add both**:

```json
{
  "domains": {
    "domain.com": "IP",        // Don't forget this!
    "*.domain.com": "IP"       // And this!
  }
}
```

---

## Testing Domain Matching

### Test if a domain matches:

```bash
# Check DNS resolution
dig @your-server-ip example.com
dig @your-server-ip www.example.com

# Check via DoH
curl "https://your-domain.com/dns-query?dns=..." \
  -H "accept: application/dns-message"

# Check what IP a website shows
curl -x http://your-server-ip:80 http://whatismyipaddress.com
```

---

## Common Mistakes

### ❌ Mistake 1: Only Wildcard
```json
{
  "domains": {
    "*.example.com": "IP"
  }
}
```
**Problem:** `example.com` won't work, only `www.example.com`

### ❌ Mistake 2: Only Exact
```json
{
  "domains": {
    "example.com": "IP"
  }
}
```
**Problem:** `www.example.com` won't work, only `example.com`

### ✅ Correct: Both Patterns
```json
{
  "domains": {
    "example.com": "IP",
    "*.example.com": "IP"
  }
}
```
**Result:** Everything works! 🎉

---

## Advanced Patterns

### Multiple Levels
Wildcard works for any subdomain level:

```json
{
  "domains": {
    "*.example.com": "IP"
  }
}
```

**Matches:**
- ✅ `www.example.com`
- ✅ `api.example.com`
- ✅ `cdn.api.example.com` (multiple levels)

### Multiple Domains, Same IP
```json
{
  "domains": {
    "site1.com": "1.2.3.4",
    "*.site1.com": "1.2.3.4",
    "site2.com": "1.2.3.4",
    "*.site2.com": "1.2.3.4",
    "site3.com": "1.2.3.4",
    "*.site3.com": "1.2.3.4"
  }
}
```

---

## Performance Notes

- ✅ Pattern matching is very fast (hash map lookup)
- ✅ Wildcards don't slow down performance
- ✅ You can add hundreds of domains without issues
- ✅ Both exact and wildcard patterns can coexist

---

## Quick Reference

| Pattern | Matches | Example |
|---------|---------|---------|
| `example.com` | Exact domain only | `example.com` ✅, `www.example.com` ❌ |
| `*.example.com` | All subdomains | `www.example.com` ✅, `example.com` ❌ |
| Both | Everything | `example.com` ✅, `www.example.com` ✅ |

---

## Need Help?

- **Issues:** https://github.com/feizezadeh/Smart-SNI-Proxy/issues
- **Discussions:** https://github.com/feizezadeh/Smart-SNI-Proxy/discussions
