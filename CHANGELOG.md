# Changelog - Smart SNI Proxy v2.0

## [2.0.0] - 2025-10-05

### 🎉 Major Release - Complete Rewrite

#### ✨ New Features

##### 👥 User Management System
- **User-Centric IP Management**
  - Create users with expiration dates
  - Multiple IPs per user (configurable max limit)
  - **FIFO (First In First Out)** IP replacement
    - When max IPs reached, oldest IP is automatically removed
    - New IP is added seamlessly
  - Unique registration link per user (`/register?token=USER_ID`)
  - IP-to-User fast lookup using `sync.Map`

- **Web Panel for User Management**
  - Clean, modern UI
  - Create/edit/delete users
  - View all registered IPs per user
  - Real-time usage statistics
  - Enable/disable users
  - Display registration links
  - Show user status (active/expired/inactive)

- **IP-Based Access Control**
  - Only registered IPs can access DoH/DoT
  - User authorization check on every request
  - Automatic expiration handling
  - Usage tracking per user

##### 🌐 DNS over HTTPS (DoH) Enhancements
- **SSL Support with Subdomain**
  - Auto-setup DoH on subdomain (e.g., `doh.example.com`)
  - Let's Encrypt SSL certificate automation
  - nginx reverse proxy on port 8443
  - HTTP/2 support

- **RFC 8484 Implementation**
  - DNS wire format (GET/POST)
  - Base64url encoding support
  - Proper error handling

- **Performance Improvements**
  - Connection pooling
  - Upstream keepalive
  - Request buffering optimization

##### 📊 Web Panel v2.0
- **Modern UI**
  - Responsive design
  - Real-time updates
  - Clean, intuitive interface
  - Persian/English support

- **Features**
  - User management dashboard
  - Statistics and metrics
  - Registration link generator
  - Bulk operations support
  - Search and filter capabilities

##### 🔧 Installation & Configuration
- **Enhanced install.sh**
  - Interactive DoH subdomain setup
  - Auto SSL certificate generation
  - nginx configuration automation
  - Firewall auto-configuration (ufw)
  - User Management menu (#6)
  - Updated menu structure

- **Configuration Updates**
  - Added `user_management` flag
  - Increased default rate limits (10→100, 20→200)
  - Better default values
  - New config validation

#### 🚀 Performance Improvements
- **Rate Limiting**
  - Increased defaults: 100 req/sec (was 10)
  - Burst: 200 (was 20)
  - Per-IP and global limits

- **Caching**
  - Optimized DNS cache
  - Better TTL handling
  - Memory efficient storage

- **Connection Management**
  - Upstream connection pooling
  - Keepalive for DoH backend
  - Better timeout handling

#### 🔐 Security Enhancements
- **Enhanced Security Headers**
  - X-Content-Type-Options: nosniff
  - X-Frame-Options: DENY
  - X-XSS-Protection: 1; mode=block
  - Referrer-Policy: no-referrer
  - Strict-Transport-Security (HSTS)
  - Content-Security-Policy

- **User Authentication**
  - SHA256 password hashing
  - Secure session management
  - Token-based user registration

- **Access Control**
  - IP-based authorization
  - User expiration checks
  - Automatic cleanup of expired users

#### 📚 Documentation
- **New Documentation Files**
  - `README.md` - Comprehensive guide (Persian/English)
  - `DOH-USAGE.md` - Complete DoH usage guide
  - `DOH-SETUP-SUMMARY.md` - DoH setup summary
  - `CHANGELOG.md` - This file
  - `test-doh.sh` - DoH testing script

- **Updated Guides**
  - Installation instructions
  - Configuration examples
  - Troubleshooting guide
  - Security best practices
  - User management workflow

#### 🛠️ Technical Changes

##### Code Structure
- **User Management**
  ```go
  type User struct {
      ID          string
      Name        string
      IPs         []string    // Multiple IPs
      MaxIPs      int         // Max allowed IPs
      CreatedAt   time.Time
      ExpiresAt   time.Time
      IsActive    bool
      Description string
      UsageCount  uint64
      LastUsed    time.Time
  }
  ```

- **FIFO Implementation**
  ```go
  func addIPToUser(userID, clientIP string) error {
      // If max IPs reached, remove oldest (FIFO)
      if len(user.IPs) >= user.MaxIPs && user.MaxIPs > 0 {
          oldestIP := user.IPs[0]
          user.IPs = user.IPs[1:]  // Remove first
          ipToUser.Delete(oldestIP)
      }

      // Add new IP
      user.IPs = append(user.IPs, clientIP)
      ipToUser.Store(clientIP, userID)
  }
  ```

- **Fast IP Lookup**
  ```go
  ipToUser sync.Map  // map[IP]userID

  func isUserAuthorized(ip string) bool {
      userIDVal, ok := ipToUser.Load(ip)
      if !ok {
          return false
      }
      // Check user status...
  }
  ```

##### API Endpoints
- `POST /api/panel/users` - Create user
- `GET /api/panel/users` - List users
- `PUT /api/panel/users/:id` - Update user
- `DELETE /api/panel/users/:id` - Delete user
- `GET /register?token=<user_id>` - User IP registration
- `POST /api/register` - Register IP for user

##### nginx Configuration
- **DoH Reverse Proxy**
  - Upstream: `http://127.0.0.1:8080`
  - Port: 8443 (SSL)
  - HTTP/2 enabled
  - Security headers
  - Connection keepalive

##### Installation Menu
```
MAIN MENU
════════════════════════════
1] Install / Upgrade
2] Uninstall
3] Show Websites
4] Add Sites
5] Remove Sites
6] User Management          ← NEW
7] View Logs
8] View Metrics
9] Restart Service
0] Exit
```

#### 🐛 Bug Fixes
- Fixed binary naming issue (smartsni vs smartSNI)
- Fixed service file path consistency
- Fixed DoH endpoint accessibility
- Fixed rate limiting calculation
- Fixed user authorization logic
- Fixed IP registration validation
- Improved error handling throughout

#### 🔄 Breaking Changes
- **Removed InviteToken System**
  - Old: InviteToken → User with single IP
  - New: User → Multiple IPs with FIFO

- **Changed User Structure**
  - Old: `User.IP` (single string)
  - New: `User.IPs` ([]string array)

- **Config Changes**
  - Added: `user_management` (bool)
  - Increased: `rate_limit_per_ip` default
  - Increased: `rate_limit_burst_ip` default

#### 📦 Installation Changes
- **Repository URL Updated**
  - Old: `github.com/Ptechgithub/smartSNI`
  - New: `github.com/feizezadeh/Smart-SNI-Proxy`

- **Default Ports**
  - SNI Proxy: 443 (unchanged)
  - DoT: 853 (unchanged)
  - DoH: 8080 local, 8443 SSL (new)
  - Web Panel: 8088 (unchanged)

#### 🎯 Migration Guide

##### From v1.x to v2.0:

1. **Backup your config:**
   ```bash
   cp /root/smartSNI/config.json /root/smartSNI/config.json.backup
   ```

2. **Run upgrade:**
   ```bash
   bash /root/smartSNI/install.sh
   # Select option 1 (Install/Upgrade)
   ```

3. **Update config.json:**
   ```json
   {
     "user_management": false,  // Add this
     "rate_limit_per_ip": 100,  // Update from 10
     "rate_limit_burst_ip": 200 // Update from 20
   }
   ```

4. **Setup DoH subdomain (optional):**
   - Create DNS A record for subdomain
   - Run install script
   - Follow DoH setup prompts

5. **Restart service:**
   ```bash
   systemctl restart sni.service
   ```

#### 📝 Post-Installation Steps

1. **Save Web Panel Password**
   - Password shown only once during installation
   - Store securely

2. **Configure DoH (if enabled)**
   - Verify subdomain DNS
   - Check SSL certificate
   - Test DoH endpoint

3. **Enable User Management (optional)**
   ```bash
   bash /root/smartSNI/install.sh
   # Select 6 (User Management)
   # Select 1 (Enable)
   ```

4. **Create First User**
   - Open Web Panel
   - Create user
   - Share registration link

#### 🔗 Useful Links
- GitHub: https://github.com/feizezadeh/Smart-SNI-Proxy
- Issues: https://github.com/feizezadeh/Smart-SNI-Proxy/issues
- Docs: See README.md and DOH-USAGE.md

#### 👏 Credits
- Original Smart SNI: [Ptechgithub](https://github.com/Ptechgithub/smartSNI)
- RFC 8484: DNS Queries over HTTPS (DoH)
- Contributors and community

---

## [1.x] - Previous Versions

See original repository for v1.x changelog.

---

**Full Changelog**: https://github.com/feizezadeh/Smart-SNI-Proxy/commits/main
