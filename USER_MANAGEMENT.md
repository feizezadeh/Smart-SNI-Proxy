# User Management System

## Overview

Smart SNI Proxy now supports user-based access control! Control who can use your DNS services with IP-based authentication and expiration dates.

## Features

✅ **IP-Based Access Control** - Only registered IPs can use DNS services
✅ **Expiration Dates** - Set validity period for each user
✅ **Invitation System** - Share registration links with users
✅ **Usage Tracking** - Monitor DNS query count and last usage
✅ **Auto-Expiration** - Automatically deactivate expired users
✅ **Web Panel Management** - Full CRUD operations via admin panel

## Quick Start

### 1. Enable User Management

Edit `config.json`:
```json
{
  "user_management": true
}
```

### 2. Create Invitation Token (Admin Panel)

```bash
curl -X POST http://YOUR_SERVER:8088/panel/api/invite/create \
  -H "X-Session-ID: YOUR_SESSION" \
  -H "Content-Type: application/json" \
  -d '{
    "valid_days": 30,
    "max_uses": 1,
    "token_expiry_days": 7
  }'
```

Response:
```json
{
  "success": true,
  "invite": {...},
  "register_url": "http://YOUR_SERVER:8088/register?token=abc123..."
}
```

### 3. Share Registration Link

Send the `register_url` to your user. They will:
1. Open the link
2. See their IP address
3. Enter their name
4. Click Register
5. Get DNS configuration details

## API Endpoints

### User Management

#### List Users
```bash
GET /panel/api/users
```

#### Create User (Manual)
```bash
POST /panel/api/users/create
{
  "ip": "1.2.3.4",
  "name": "John Doe",
  "description": "Home connection",
  "valid_days": 30
}
```

#### Extend User
```bash
POST /panel/api/users/extend
{
  "user_id": "abc123",
  "days": 30
}
```

#### Deactivate User
```bash
POST /panel/api/users/deactivate
{
  "user_id": "abc123"
}
```

#### Delete User
```bash
POST /panel/api/users/delete
{
  "user_id": "abc123"
}
```

### Invitation Management

#### Create Invite Token
```bash
POST /panel/api/invite/create
{
  "valid_days": 30,        // Days of DNS access for user
  "max_uses": 1,           // Max registrations (0 = unlimited)
  "token_expiry_days": 7   // Days until token expires
}
```

#### List Invites
```bash
GET /panel/api/invite/list
```

## Registration Flow

1. **Admin creates invite token**
   - Sets access duration (e.g., 30 days)
   - Sets token max uses (e.g., 1 time)
   - Gets registration URL

2. **User opens registration link**
   ```
   http://YOUR_SERVER:8088/register?token=TOKEN_HERE
   ```

3. **Registration page shows**
   - User's IP address (auto-detected)
   - Access duration
   - Token expiration date
   - Name input field
   - Description field (optional)

4. **User submits registration**
   - System validates token
   - Checks if IP already registered
   - Creates user with expiration date
   - Returns DNS configuration

5. **User gets DNS settings**
   ```
   DoH: https://your-domain.com/dns-query
   DoT: your-domain.com:853
   ```

## Access Control

When `user_management: true`:
- DoH requests: Only authorized IPs allowed
- DoT connections: Only authorized IPs allowed
- SNI Proxy: Not affected (domain-based)

When `user_management: false`:
- All IPs allowed (default behavior)

## User Data Structure

```json
{
  "id": "a1b2c3d4",
  "ip": "1.2.3.4",
  "name": "John Doe",
  "description": "Home connection",
  "created_at": "2025-10-04T10:00:00Z",
  "expires_at": "2025-11-04T10:00:00Z",
  "is_active": true,
  "usage_count": 1250,
  "last_used": "2025-10-04T15:30:00Z"
}
```

## Expiration System

- **Background Checker**: Runs every hour
- **Auto-Deactivation**: Expired users are automatically deactivated
- **No Service**: Deactivated users cannot use DNS services
- **Reactivation**: Extend expiration date to reactivate

## Security Notes

⚠️ **IP-Based Authentication**
- Users can only register from their actual IP
- IP changes require new registration
- No password needed (IP is the credential)

⚠️ **Token Security**
- Tokens are random 32-character hex strings
- Set short expiry for one-time use tokens
- Monitor token usage via API

## Example Usage

### Scenario: ISP Providing DNS Service

1. **Customer requests access**
2. **Admin creates 30-day token** with max 1 use
3. **Customer registers** from their home IP
4. **Customer configures** DNS on their devices
5. **After 30 days**, access automatically expires
6. **Customer requests renewal**
7. **Admin extends** for another 30 days

## Monitoring

Check user activity:
```bash
curl http://YOUR_SERVER:8088/panel/api/users \
  -H "X-Session-ID: YOUR_SESSION"
```

View:
- Total users
- Active/inactive status
- Expiration dates
- Usage statistics
- Last access time

## Troubleshooting

### User can't access DNS
1. Check if user is active: `GET /panel/api/users`
2. Verify expiration date hasn't passed
3. Confirm IP address matches registration
4. Check `user_management` is enabled in config

### Token not working
1. Verify token hasn't expired
2. Check max uses not exceeded
3. Ensure token is active
4. List all tokens: `GET /panel/api/invite/list`

## Integration Examples

### Python
```python
import requests

# Create invite
response = requests.post(
    'http://YOUR_SERVER:8088/panel/api/invite/create',
    headers={'X-Session-ID': session_id},
    json={'valid_days': 30, 'max_uses': 1, 'token_expiry_days': 7}
)

invite_url = response.json()['register_url']
print(f"Send this to user: {invite_url}")
```

### Bash
```bash
#!/bin/bash

# Create invite and extract URL
INVITE=$(curl -s -X POST http://YOUR_SERVER:8088/panel/api/invite/create \
  -H "X-Session-ID: $SESSION_ID" \
  -H "Content-Type: application/json" \
  -d '{"valid_days":30,"max_uses":1,"token_expiry_days":7}')

URL=$(echo $INVITE | jq -r '.register_url')
echo "Registration link: $URL"

# Send to user (email, SMS, etc.)
```

## Future Enhancements

- [ ] Email notifications before expiration
- [ ] Bandwidth limits per user
- [ ] User groups and permissions
- [ ] OAuth/SSO integration
- [ ] Mobile app for registration
- [ ] User self-service portal

## License

Part of Smart SNI Proxy v2.0
