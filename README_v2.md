# Smart SNI Proxy v2.0 - Complete Edition

A high-performance DNS proxy server with SNI routing, supporting DNS-over-HTTPS (DoH), DNS-over-TLS (DoT), and intelligent caching.

## 🆕 What's New in v2.0

### Security Enhancements
- ✅ **Bearer Token Authentication** for DoH/DoT endpoints
- ✅ **Per-IP Rate Limiting** to prevent abuse
- ✅ **Domain Blocking** support
- ✅ **Enhanced TLS Configuration** (TLS 1.2/1.3 with strong cipher suites)
- ✅ **Security Headers** (X-Frame-Options, CSP, etc.)
- ✅ **Input Validation** for DNS queries

### Monitoring & Observability
- ✅ **Structured JSON Logging** with configurable log levels (debug, info, warn, error)
- ✅ **Prometheus Metrics** endpoint (`/metrics/prometheus`)
- ✅ **Health Check** endpoint (`/health`)
- ✅ **Real-time Statistics** tracking:
  - DoH queries
  - DoT queries
  - SNI connections
  - Cache hits/misses
  - Error counts

### Performance Improvements
- ✅ **DNS Response Caching** with configurable TTL
- ✅ **Multiple Upstream DoH Servers** with automatic failover
- ✅ **Connection Pooling** for upstream queries
- ✅ **Automatic Cache Cleanup** to prevent memory leaks
- ✅ **Panic Recovery** in all handlers

### Configuration & Management
- ✅ **Enhanced Config Validation** at startup
- ✅ **Hot Reload** via `/admin/reload` endpoint (no restart required)
- ✅ **Thread-safe Configuration** updates
- ✅ **Environment Variable** support for DoH upstream
- ✅ **Flexible Domain Matching** with wildcard support

---

## 📋 Configuration

### config.json

```json
{
  "host": "your.domain.com",
  "domains": {
    "*.youtube.com": "1.2.3.4",
    "*.google.com": "1.2.3.4"
  },
  "upstream_doh": [
    "https://1.1.1.1/dns-query",
    "https://8.8.8.8/dns-query"
  ],
  "enable_auth": false,
  "auth_tokens": [
    "your-secret-token-here"
  ],
  "cache_ttl": 300,
  "rate_limit_per_ip": 10,
  "rate_limit_burst_ip": 20,
  "log_level": "info",
  "trusted_proxies": [],
  "blocked_domains": [
    "*.badsite.com"
  ],
  "metrics_enabled": true
}
```

### Configuration Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `host` | string | **required** | Your server's hostname |
| `domains` | object | **required** | Domain patterns to proxy (supports wildcards) |
| `upstream_doh` | array | `["https://1.1.1.1/dns-query"]` | Upstream DoH servers (with failover) |
| `enable_auth` | bool | `false` | Enable Bearer token authentication |
| `auth_tokens` | array | `[]` | Valid authentication tokens |
| `cache_ttl` | int | `300` | DNS cache TTL in seconds |
| `rate_limit_per_ip` | int | `10` | Max requests per second per IP |
| `rate_limit_burst_ip` | int | `20` | Burst limit for rate limiting |
| `log_level` | string | `"info"` | Logging level (debug/info/warn/error) |
| `trusted_proxies` | array | `[]` | List of trusted proxy IPs |
| `blocked_domains` | array | `[]` | Domain patterns to block |
| `metrics_enabled` | bool | `true` | Enable metrics endpoints |

---

## 🚀 Installation

### Auto Install (Recommended)

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/feizezadeh/Smart-SNI-Proxy/install.sh)
```

### Manual Installation

1. **Install Dependencies**
```bash
apt update
apt install nginx certbot python3-certbot-nginx
snap install go --classic
```

2. **Clone Repository**
```bash
git clone https://github.com/feizezadeh/Smart-SNI-Proxy.git /root/Smart-SNI-Proxy
cd /root/Smart-SNI-Proxy
```

3. **Configure**
```bash
# Edit config.json with your settings
nano config.json

# Update nginx config
nano nginx.conf
```

4. **Obtain SSL Certificate**
```bash
certbot --nginx -d your.domain.com
```

5. **Build & Run**
```bash
go build -o smartsni main.go
./smartsni
```

### Running as Service

```bash
# Create systemd service
cat > /etc/systemd/system/sni.service <<EOF
[Unit]
Description=Smart SNI Proxy v2.0
After=network.target

[Service]
User=root
WorkingDirectory=/root/smartSNI
ExecStart=/root/Smart-SNI-Proxy/smartsni
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
systemctl daemon-reload
systemctl enable sni.service
systemctl start sni.service
```

---

## 📡 API Endpoints

### DoH Endpoints

#### DNS Query
- **URL**: `https://your.domain.com/dns-query`
- **Methods**: `GET`, `POST`
- **GET Format**: `/dns-query?dns=<base64url-encoded-query>`
- **POST Format**: Binary DNS message in body
- **Headers**:
  - `Content-Type: application/dns-message`
  - `Authorization: Bearer <token>` (if auth enabled)

### Monitoring Endpoints

#### Health Check
```bash
curl http://127.0.0.1:8080/health
```
Response:
```json
{
  "status": "healthy",
  "uptime": 3600.5,
  "version": "2.0"
}
```

#### Metrics (JSON)
```bash
curl http://127.0.0.1:8080/metrics
```
Response:
```json
{
  "doh_queries": 12345,
  "dot_queries": 6789,
  "sni_connections": 9876,
  "cache_hits": 5432,
  "cache_misses": 1234,
  "errors": 42
}
```

#### Prometheus Metrics
```bash
curl http://127.0.0.1:8080/metrics/prometheus
```
Response:
```
# HELP smartsni_doh_queries_total Total number of DoH queries
# TYPE smartsni_doh_queries_total counter
smartsni_doh_queries_total 12345
...
```

### Admin Endpoints

#### Reload Configuration
```bash
curl -X POST http://127.0.0.1:8080/admin/reload \
  -H "Authorization: Bearer your-token"
```

---

## 🔒 Security Features

### Authentication

Enable authentication in `config.json`:
```json
{
  "enable_auth": true,
  "auth_tokens": ["secret-token-1", "secret-token-2"]
}
```

Use with requests:
```bash
curl https://your.domain.com/dns-query \
  -H "Authorization: Bearer secret-token-1" \
  --data-binary @query.bin
```

### Rate Limiting

Two-tier rate limiting:
1. **Global**: 50 req/s with burst of 100
2. **Per-IP**: Configurable (default: 10 req/s, burst 20)

### Domain Blocking

Block unwanted domains:
```json
{
  "blocked_domains": [
    "*.ads.example.com",
    "tracker.bad.com"
  ]
}
```

---

## 📊 Monitoring with Prometheus

Example `prometheus.yml`:
```yaml
scrape_configs:
  - job_name: 'smartsni'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics/prometheus'
```

---

## 🔧 Troubleshooting

### View Logs
```bash
# If running as service
journalctl -u sni.service -f

# If running manually
./smartsni 2>&1 | tee smartsni.log
```

### Enable Debug Logging
```json
{
  "log_level": "debug"
}
```

### Check Service Status
```bash
systemctl status sni.service
```

### Test DoH Endpoint
```bash
# Using curl
curl -H "accept: application/dns-message" \
  "https://your.domain.com/dns-query?dns=AAABAAABAAAAAAAAA3d3dwdleGFtcGxlA2NvbQAAAQAB"

# Using kdig
kdig @your.domain.com +https example.com
```

---

## 🎯 Performance Tuning

### Increase Cache Size
```json
{
  "cache_ttl": 600
}
```

### Add More Upstream Servers
```json
{
  "upstream_doh": [
    "https://1.1.1.1/dns-query",
    "https://1.0.0.1/dns-query",
    "https://8.8.8.8/dns-query",
    "https://8.8.4.4/dns-query"
  ]
}
```

### Optimize Rate Limits
```json
{
  "rate_limit_per_ip": 20,
  "rate_limit_burst_ip": 50
}
```

---

## 📜 Changelog

### v2.0 (Current)
- Added structured logging with JSON output
- Implemented Prometheus metrics
- Added DNS caching mechanism
- Multiple upstream DoH servers with failover
- Per-IP rate limiting
- Bearer token authentication
- Domain blocking support
- Health check endpoint
- Configuration hot reload
- Enhanced security headers
- Panic recovery in all handlers
- Improved error handling

### v1.0
- Basic DoH/DoT/SNI proxy
- Simple configuration
- Let's Encrypt integration

---

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

---

## 📄 License

This project is open-source and available under the [MIT License](LICENSE).

---

## 🙏 Credits

- Original author: [Ptechgithub](https://github.com/Ptechgithub)
- Enhanced by: Community contributors

---

## 📞 Support

For issues and questions:
- GitHub Issues: https://github.com/Ptechgithub/smartSNI/issues
- Discussions: https://github.com/Ptechgithub/smartSNI/discussions
