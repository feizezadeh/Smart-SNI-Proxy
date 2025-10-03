# 🌐 Web Panel - User Guide

Smart SNI Proxy v2.0 includes a powerful web-based control panel for easy management of your DNS/SNI proxy server.

## 🚀 Quick Start

### 1. Enable Web Panel

Edit `config.json`:
```json
{
  "web_panel_enabled": true,
  "web_panel_username": "admin",
  "web_panel_password": "YOUR_SHA256_HASH_HERE",
  "web_panel_port": 8088
}
```

### 2. Generate Password Hash

```bash
./generate_password.sh mypassword
```

Or manually:
```bash
echo -n "mypassword" | sha256sum
```

**Default credentials:**
- Username: `admin`
- Password: `admin` (hash: `8c6976e5b5410415bde908bd4dee15dfb167a9c873fc4bb8a81f6f2ab448a918`)

⚠️ **IMPORTANT:** Change the default password immediately after first login!

### 3. Access Web Panel

Open your browser and navigate to:
```
http://your-server-ip:8088/panel
```

Or locally:
```
http://localhost:8088/panel
```

---

## 📋 Features

### Dashboard Overview
- **Real-time Metrics**
  - DoH queries counter
  - DoT queries counter
  - SNI connections counter
  - Cache hit rate percentage

- **System Information**
  - Server version
  - Uptime
  - Health status

### Domain Management
- ✅ Add new domains with IP mapping
- ✅ Remove existing domains
- ✅ View all configured domains
- ✅ Automatic wildcard pattern (`*.domain.com`)

### Configuration
- ✅ Hot reload configuration
- ✅ No server restart required
- ✅ Changes applied instantly

### Security
- ✅ Session-based authentication
- ✅ 24-hour session timeout
- ✅ Automatic session cleanup
- ✅ Password hashing (SHA256)
- ✅ Login attempt logging

---

## 🎨 Interface Guide

### Login Page
Simple and secure login with username/password authentication.

![Login](https://via.placeholder.com/800x400?text=Login+Page)

### Dashboard
Clean, modern interface with:
- Statistics cards showing real-time metrics
- Domain management panel
- System information panel
- Activity logs (upcoming feature)

![Dashboard](https://via.placeholder.com/800x400?text=Dashboard)

---

## 🔒 Security Best Practices

### 1. Change Default Password
```bash
# Generate new password hash
./generate_password.sh your-strong-password

# Update config.json with new hash
```

### 2. Restrict Access
Use firewall to limit access to trusted IPs:
```bash
# UFW example
ufw allow from 192.168.1.0/24 to any port 8088

# iptables example
iptables -A INPUT -p tcp --dport 8088 -s 192.168.1.0/24 -j ACCEPT
iptables -A INPUT -p tcp --dport 8088 -j DROP
```

### 3. Use HTTPS (Recommended)
Put the web panel behind nginx with SSL:

```nginx
server {
    listen 443 ssl http2;
    server_name panel.your-domain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://127.0.0.1:8088;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 4. Use Strong Passwords
- Minimum 12 characters
- Mix of uppercase, lowercase, numbers, symbols
- Don't reuse passwords from other services

---

## 📡 API Endpoints

The web panel uses REST API endpoints:

### Authentication
- `POST /panel/api/login` - Login with credentials
- `POST /panel/api/logout` - Logout and destroy session
- `GET /panel/api/validate` - Validate session

### Data
- `GET /panel/api/metrics` - Get system metrics
- `GET /panel/api/health` - Get health status
- `GET /panel/api/domains` - List all domains

### Management
- `POST /panel/api/domains/add` - Add new domain
- `POST /panel/api/domains/remove` - Remove domain
- `POST /panel/api/reload` - Reload configuration

### Example API Call
```bash
# Login
curl -X POST http://localhost:8088/panel/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}'

# Response: {"session_id":"...","username":"admin"}

# Get metrics (use session_id from login)
curl http://localhost:8088/panel/api/metrics \
  -H "X-Session-ID: your-session-id"
```

---

## 🔧 Troubleshooting

### Web Panel Not Starting

**Check 1:** Is it enabled?
```bash
cat config.json | grep web_panel_enabled
# Should show: "web_panel_enabled": true
```

**Check 2:** Are credentials configured?
```bash
cat config.json | grep web_panel_username
cat config.json | grep web_panel_password
```

**Check 3:** Port conflict?
```bash
netstat -tulpn | grep 8088
# If port is in use, change web_panel_port in config.json
```

**Check 4:** View logs
```bash
journalctl -u sni.service -f | grep "web panel"
```

### Cannot Login

**Issue:** "Invalid credentials"
- Verify password hash is correct
- Check username matches config
- Password is case-sensitive

**Issue:** "Invalid session"
- Session expired (24 hour timeout)
- Clear browser localStorage and login again

### Domains Not Updating

**Solution:** Click "Reload Configuration" button in web panel
```bash
# Or via API
curl -X POST http://localhost:8088/panel/api/reload \
  -H "X-Session-ID: your-session-id"
```

---

## ⚙️ Configuration Reference

### Complete Web Panel Settings

```json
{
  "web_panel_enabled": true,
  "web_panel_username": "admin",
  "web_panel_password": "SHA256_HASH_HERE",
  "web_panel_port": 8088
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `web_panel_enabled` | bool | `false` | Enable/disable web panel |
| `web_panel_username` | string | - | Admin username |
| `web_panel_password` | string | - | SHA256 password hash |
| `web_panel_port` | int | `8088` | Web panel listen port |

---

## 🎯 Tips & Tricks

### 1. Bookmark the Panel
```
http://your-server-ip:8088/panel
```

### 2. Use Different Port
If 8088 is blocked or in use:
```json
{
  "web_panel_port": 9000
}
```

### 3. Multiple Admin Users
Currently supports single user. For multiple users, use nginx auth:
```nginx
location /panel {
    auth_basic "Restricted";
    auth_basic_user_file /etc/nginx/.htpasswd;
    proxy_pass http://127.0.0.1:8088;
}
```

### 4. Monitoring Integration
Use the metrics API for external monitoring:
```bash
# Add to Prometheus
curl http://localhost:8088/panel/api/metrics \
  -H "X-Session-ID: $SESSION_ID"
```

---

## 📊 Screenshots

### Statistics View
Real-time counters update every 5 seconds automatically.

### Domain Management
Easy add/remove interface with validation:
- Automatic wildcard prefix (`*.` added automatically)
- IP address validation
- Duplicate detection

### Mobile Responsive
Works great on tablets and phones!

---

## 🔄 Updates & Migration

### From v1.0 to v2.0
1. Add web panel settings to `config.json`
2. Generate password hash
3. Restart service

### Updating Password
1. Generate new hash: `./generate_password.sh newpassword`
2. Update `web_panel_password` in `config.json`
3. Click "Reload Configuration" in web panel (or restart service)

---

## 💡 Future Features

Coming soon:
- [ ] Real-time log viewer
- [ ] Multiple user accounts
- [ ] 2FA authentication
- [ ] Statistics graphs/charts
- [ ] Configuration backup/restore
- [ ] Blocked domains management
- [ ] Upstream DoH server management
- [ ] Rate limiting configuration

---

## 🆘 Support

- **Issues:** https://github.com/feizezadeh/Smart-SNI-Proxy/issues
- **Discussions:** https://github.com/feizezadeh/Smart-SNI-Proxy/discussions
- **Documentation:** https://github.com/feizezadeh/Smart-SNI-Proxy

---

## 📄 License

MIT License - Same as Smart SNI Proxy main project

---

**Note:** The default password hash (`8c6976e5b5410415bde908bd4dee15dfb167a9c873fc4bb8a81f6f2ab448a918`) is for the password `admin`. **Change it immediately after installation!**
