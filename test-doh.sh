#!/bin/bash

# تست DoH با استفاده از dig
# Test DoH using dig command

echo "🧪 Testing DoH endpoint..."
echo ""

# Method 1: Using dig with DoH support (if available)
if command -v dig >/dev/null 2>&1; then
    echo "📡 Method 1: Using dig with DoH"
    echo "dig @89.47.113.135 -p 8080 google.com"
    dig @89.47.113.135 -p 8080 google.com +short
    echo ""
fi

# Method 2: Using curl with proper DNS wire format
echo "📡 Method 2: Using curl with DoH (RFC 8484)"
echo ""

# Create a DNS query for google.com A record using kdig or similar tool
# For now, we'll use a pre-encoded query for google.com
# This is base64url encoded DNS wire format query for "google.com A"

# Example DNS query (you need to generate this properly)
# For testing, we can use Google's public DoH to see the format:
echo "Example: Testing with cloudflare DoH first to understand format:"
curl -s -H 'accept: application/dns-message' 'https://1.1.1.1/dns-query?dns=AAABAAABAAAAAAAAB2V4YW1wbGUDY29tAAABAAE' | xxd | head -5
echo ""

echo "ℹ️  DoH requires RFC 8484 format:"
echo "   - GET: ?dns=<base64url-encoded-dns-wire-format>"
echo "   - POST: Raw DNS wire format in body"
echo ""
echo "📝 To use this DoH server, configure your DoH client with:"
echo "   URL: https://doh.dnsoverhttps.site:8443/dns-query"
echo ""
echo "⚠️  Current rate limit: 10 req/sec per IP (burst 20)"
echo "   Consider increasing for production use"
