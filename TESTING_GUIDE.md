# Testing Guide - Smart SNI Proxy v2.0

## ✅ Features Implemented

### 1. **User Management System**
- IP-based access control
- Invitation token system
- Expiration dates with auto-deactivation
- Usage tracking

### 2. **Improved Domain Management**
- Auto-detect server IP
- Add both exact and wildcard patterns automatically
- Clean domain input

### 3. **Web Panel Enhancements**
- User Management section added
- Create invite links
- Extend/Delete users
- Real-time updates

## 🧪 Testing on Server

### SSH to Server
```bash
ssh root@89.47.113.135
# Password: Mefe160502@136525
```

### 1. Check Service Status
```bash
systemctl status sni.service
journalctl -u sni.service -n 20
```

### 2. Check Configuration
```bash
cat /root/smartSNI/config.json
```

**Expected config.json:**
```json
{
  "host": "your-domain.com",
  "domains": {
    "*.youtube.com": "89.47.113.135",
    "*.google.com": "89.47.113.135"
  },
  "web_panel_enabled": true,
  "web_panel_port": 8088,
  "user_management": false
}
```

### 3. Test Web Panel Access
```bash
# Check if port 8088 is listening
ss -tulnp | grep 8088

# Test locally
curl http://localhost:8088/panel
```

**Access from browser:**
- URL: http://89.47.113.135:8088/panel
- Username: `admin`
- Password: `admin`

### 4. Test Domain Addition

**In Web Panel:**
1. Login to admin panel
2. Go to "Managed Domains" section
3. Enter domain name: `facebook.com`
4. Click "Add"
5. Check that both `facebook.com` and `*.facebook.com` are added

**Verify in config:**
```bash
cat /root/smartSNI/config.json | jq '.domains'
```

Should show:
```json
{
  "facebook.com": "89.47.113.135",
  "*.facebook.com": "89.47.113.135",
  ...
}
```

### 5. Test User Management

#### Enable User Management
```bash
cd /root/smartSNI
nano config.json
# Change: "user_management": true
systemctl restart sni.service
```

#### Create Invite Link (Web Panel)
1. Go to "User Management" section
2. Click "Create Invite Link"
3. Enter:
   - Access duration: `30` days
   - Max uses: `1`
   - Token expiry: `7` days
4. Copy the generated link

#### Test Registration
1. Open the invite link in browser (from different device or incognito)
2. Enter name: `Test User`
3. Click "Register"
4. Should see DNS configuration

#### Verify User Created
**In Web Panel:**
- Refresh "User Management" section
- Should see new user with:
  - IP address
  - Name: "Test User"
  - Expiry date: 30 days from now
  - Status: Active
  - Usage: 0

**Command line:**
```bash
# Check logs for user creation
journalctl -u sni.service -n 50 | grep "user created"
```

### 6. Test User Access Control

**With user_management: true**

Test DoH query:
```bash
# From registered IP (should work)
curl -H "accept: application/dns-json" \
  "http://89.47.113.135:8080/dns-query?name=google.com&type=A"

# From non-registered IP (should fail with 403)
# Try from different server or VPN
```

### 7. Test User Operations

#### Extend User
1. In Web Panel, click "Extend" on user
2. Enter days: `30`
3. Verify expiry date updated

#### Delete User
1. Click "Delete" on user
2. Confirm deletion
3. Verify user removed from list

### 8. Test API Endpoints

#### List Users
```bash
SESSION_ID="YOUR_SESSION_ID"  # Get from browser DevTools

curl -X GET http://89.47.113.135:8088/panel/api/users \
  -H "X-Session-ID: $SESSION_ID"
```

#### Create Invite Token
```bash
curl -X POST http://89.47.113.135:8088/panel/api/invite/create \
  -H "X-Session-ID: $SESSION_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "valid_days": 30,
    "max_uses": 1,
    "token_expiry_days": 7
  }'
```

Response should include `register_url`.

## 🐛 Troubleshooting

### Service Won't Start

**Check logs:**
```bash
journalctl -u sni.service -n 50 --no-pager
```

**Common issues:**

1. **Invalid IP in config**
```bash
# Error: invalid IP address for domain *.youtube.com: <YOUR_IP>
# Fix:
sed -i 's/<YOUR_IP>/89.47.113.135/g' /root/smartSNI/config.json
systemctl restart sni.service
```

2. **Invalid JSON**
```bash
# Validate JSON
cat /root/smartSNI/config.json | jq '.'
```

3. **Missing fields**
```bash
# Ensure config has all required fields
cat /root/smartSNI/config.json | jq 'keys'
```

### Web Panel Not Accessible

**Check port:**
```bash
ss -tulnp | grep 8088
```

**Check firewall:**
```bash
ufw status
# If active, allow port:
ufw allow 8088/tcp
```

**Check logs:**
```bash
journalctl -u sni.service | grep "web panel"
```

### Users Not Loading

**Check user_management flag:**
```bash
cat /root/smartSNI/config.json | jq '.user_management'
# Should be true or false (not missing)
```

**Check API response:**
```bash
# Get session ID from browser (F12 > Application > LocalStorage > sessionId)
curl http://89.47.113.135:8088/panel/api/users \
  -H "X-Session-ID: YOUR_SESSION_ID"
```

## 📋 Quick Fix Commands

### Rebuild Service
```bash
cd /root/smartSNI
/usr/local/go/bin/go build -o smartsni main.go
systemctl restart sni.service
systemctl status sni.service
```

### Fix Config
```bash
cd /root/smartSNI

# Get server IP
SERVER_IP=$(hostname -I | awk '{print $1}')

# Fix all <YOUR_IP> placeholders
sed -i "s/<YOUR_IP>/$SERVER_IP/g" config.json

# Fix <YOUR_HOST> if needed
sed -i 's/<YOUR_HOST>/your-domain.com/g' config.json

# Validate
cat config.json | jq '.'

# Restart
systemctl restart sni.service
```

### View Logs Live
```bash
journalctl -u sni.service -f
```

### Test Locally
```bash
# Test DoH
curl http://localhost:8080/dns-query?name=google.com

# Test Web Panel
curl http://localhost:8088/panel

# Check ports
ss -tulnp | grep -E '8080|8088|443|853'
```

## ✅ Success Criteria

1. **Service Running**
   - `systemctl status sni.service` shows "active (running)"
   - No errors in logs

2. **Web Panel Accessible**
   - Can login at http://89.47.113.135:8088/panel
   - Dashboard loads with metrics
   - Can add domains
   - Can see User Management section

3. **Domain Management Works**
   - Adding "example.com" creates both "example.com" and "*.example.com"
   - No IP input required
   - Config updates correctly

4. **User Management Works** (if enabled)
   - Can create invite links
   - Registration page works
   - Users appear in list
   - Can extend/delete users
   - Access control enforced

## 📝 Notes

- **Default user_management**: `false` (all IPs allowed)
- **Web Panel Credentials**: admin / admin (hash in config)
- **Server IP**: Auto-detected from existing domains
- **Domain Patterns**: Both exact and wildcard added automatically

## 🚀 Next Steps

1. Test all features manually
2. Enable `user_management: true` for testing
3. Create test user via invite link
4. Verify access control works
5. Test from different IPs if possible

## 📞 Support

If issues persist:
1. Check logs: `journalctl -u sni.service -n 100`
2. Verify config: `cat /root/smartSNI/config.json | jq '.'`
3. Check network: `ss -tulnp | grep -E '8080|8088'`
4. Rebuild: `cd /root/smartSNI && /usr/local/go/bin/go build -o smartsni main.go`
