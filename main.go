package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/miekg/dns"
	"github.com/valyala/fasthttp"
	"golang.org/x/time/rate"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// ======================== Globals ========================

var (
	// Reusable buffer to reduce allocations in io.Copy and reads
	BufferPool = sync.Pool{
		New: func() interface{} {
			// 16KB tends to work well for TLS/DNS framing
			return make([]byte, 16*1024)
		},
	}

	config      atomic.Value // *Config - thread-safe config
	limiter     *rate.Limiter
	ipLimiters  sync.Map // map[string]*rate.Limiter - per-IP rate limiting
	users       sync.Map // map[userID]*User - user management
	ipToUser    sync.Map // map[IP]userID - IP to User mapping for fast lookup
	dohURL      = "https://1.1.1.1/dns-query"
	dohUpstream atomic.Value // []string - multiple upstream servers
	dohClient   = &http.Client{
		Timeout: 4 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  false,
			TLSHandshakeTimeout: 3 * time.Second,
		},
	}
	defaultTTL uint32 = 3600
	logger     *slog.Logger

	// Metrics
	metrics = &Metrics{
		dohQueries:    0,
		dotQueries:    0,
		sniConnections: 0,
		cacheHits:     0,
		cacheMisses:   0,
		errors:        0,
	}

	// DNS Cache
	dnsCache     sync.Map // map[string]*CacheEntry
	authTokens   sync.Map // map[string]bool - valid auth tokens

	// Web Panel Sessions
	webSessions sync.Map // map[string]*Session - session_id -> Session
)

// ======================== Structs ========================

// Config holds the main configuration
type Config struct {
	Host              string            `json:"host"`
	Domains           map[string]string `json:"domains"` // pattern -> IP (supports exact or "*.example.com")
	SNIPort           int               `json:"sni_port,omitempty"`           // SNI proxy port (default 443)
	DNSEnabled        bool              `json:"dns_enabled,omitempty"`        // Enable standard DNS on port 53
	UpstreamDOH       []string          `json:"upstream_doh,omitempty"`
	AuthTokens        []string          `json:"auth_tokens,omitempty"`
	EnableAuth        bool              `json:"enable_auth,omitempty"`
	CacheTTL          int               `json:"cache_ttl,omitempty"`          // seconds
	RateLimitPerIP    int               `json:"rate_limit_per_ip,omitempty"`  // requests per second
	RateLimitBurstIP  int               `json:"rate_limit_burst_ip,omitempty"`
	LogLevel          string            `json:"log_level,omitempty"` // debug, info, warn, error
	TrustedProxies    []string          `json:"trusted_proxies,omitempty"`
	BlockedDomains    []string          `json:"blocked_domains,omitempty"`
	MetricsEnabled    bool              `json:"metrics_enabled,omitempty"`
	WebPanelEnabled   bool              `json:"web_panel_enabled,omitempty"`
	WebPanelUsername  string            `json:"web_panel_username,omitempty"`
	WebPanelPassword  string            `json:"web_panel_password,omitempty"` // SHA256 hash
	WebPanelPort      int               `json:"web_panel_port,omitempty"`
	UserManagement    bool              `json:"user_management,omitempty"`    // Enable user-based access control
}

// User represents a registered user with IP-based access
type User struct {
	ID          string    `json:"id"`           // Unique user ID (also used as token)
	Name        string    `json:"name"`         // User's name/identifier
	IPs         []string  `json:"ips"`          // List of registered IPs (FIFO)
	MaxIPs      int       `json:"max_ips"`      // Maximum number of IPs allowed
	CreatedAt   time.Time `json:"created_at"`   // Registration time
	ExpiresAt   time.Time `json:"expires_at"`   // Expiration time
	IsActive    bool      `json:"is_active"`    // Active status
	Description string    `json:"description"`  // Optional description
	UsageCount  uint64    `json:"usage_count"`  // Number of DNS queries made
	LastUsed    time.Time `json:"last_used"`    // Last query time
}


// Metrics holds runtime statistics
type Metrics struct {
	dohQueries     uint64
	dotQueries     uint64
	sniConnections uint64
	cacheHits      uint64
	cacheMisses    uint64
	errors         uint64
	mu             sync.RWMutex
}

// CacheEntry represents a cached DNS response
type CacheEntry struct {
	Response  []byte
	ExpiresAt time.Time
	mu        sync.RWMutex
}

// Session represents a web panel session
type Session struct {
	Username  string
	CreatedAt time.Time
	ExpiresAt time.Time
	mu        sync.RWMutex
}

func LoadConfig(filename string) (*Config, error) {
	var c Config
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}

	// Set defaults
	if len(c.UpstreamDOH) == 0 {
		c.UpstreamDOH = []string{"https://1.1.1.1/dns-query", "https://8.8.8.8/dns-query"}
	}
	if c.CacheTTL == 0 {
		c.CacheTTL = 300 // 5 minutes default
	}
	if c.RateLimitPerIP == 0 {
		c.RateLimitPerIP = 10
	}
	if c.RateLimitBurstIP == 0 {
		c.RateLimitBurstIP = 20
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	if c.WebPanelPort == 0 {
		c.WebPanelPort = 8088
	}

	// Validate config
	if err := validateConfig(&c); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &c, nil
}

func SaveConfig(filename string, c *Config) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func validateConfig(c *Config) error {
	if c.Host == "" {
		return errors.New("host cannot be empty")
	}
	if len(c.Domains) == 0 {
		return errors.New("domains cannot be empty")
	}
	for pattern, ip := range c.Domains {
		if pattern == "" {
			return errors.New("domain pattern cannot be empty")
		}
		if net.ParseIP(ip) == nil {
			return fmt.Errorf("invalid IP address for domain %s: %s", pattern, ip)
		}
	}
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s", c.LogLevel)
	}
	return nil
}

func getConfig() *Config {
	return config.Load().(*Config)
}

func reloadConfig(filename string) error {
	newConfig, err := LoadConfig(filename)
	if err != nil {
		return err
	}

	config.Store(newConfig)

	// Update upstream servers
	dohUpstream.Store(newConfig.UpstreamDOH)

	// Update auth tokens
	authTokens.Range(func(key, value interface{}) bool {
		authTokens.Delete(key)
		return true
	})
	for _, token := range newConfig.AuthTokens {
		authTokens.Store(token, true)
	}

	logger.Info("configuration reloaded successfully")
	return nil
}

// ======================== Utilities ========================

func initLogger(level string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(handler)
}

func isIPv4(ip net.IP) bool { return ip.To4() != nil }

func trimDot(s string) string { return strings.TrimSuffix(s, ".") }

func countDots(s string) int { return strings.Count(s, ".") }

func getClientIP(ctx *fasthttp.RequestCtx) string {
	// Check X-Forwarded-For header
	xff := string(ctx.Request.Header.Peek("X-Forwarded-For"))
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	// Check X-Real-IP header
	xri := string(ctx.Request.Header.Peek("X-Real-IP"))
	if xri != "" {
		return xri
	}
	return ctx.RemoteIP().String()
}

func getIPLimiter(ip string) *rate.Limiter {
	cfg := getConfig()
	val, exists := ipLimiters.Load(ip)
	if !exists {
		limiter := rate.NewLimiter(rate.Limit(cfg.RateLimitPerIP), cfg.RateLimitBurstIP)
		ipLimiters.Store(ip, limiter)
		return limiter
	}
	return val.(*rate.Limiter)
}

func checkAuth(ctx *fasthttp.RequestCtx) bool {
	cfg := getConfig()
	if !cfg.EnableAuth {
		return true
	}

	authHeader := string(ctx.Request.Header.Peek("Authorization"))
	if authHeader == "" {
		return false
	}

	// Support Bearer token
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		_, exists := authTokens.Load(token)
		return exists
	}

	return false
}

func isDomainBlocked(domain string) bool {
	cfg := getConfig()
	for _, blocked := range cfg.BlockedDomains {
		if matches(domain, blocked) {
			return true
		}
	}
	return false
}

// ======================== Web Panel Auth ========================

func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ======================== User Management Functions ========================

func generateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func generateUserID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Check if user is authorized based on IP
func isUserAuthorized(ip string) bool {
	cfg := getConfig()
	if !cfg.UserManagement {
		return true // User management disabled, allow all
	}

	// Fast lookup using ipToUser map
	userIDVal, ok := ipToUser.Load(ip)
	if !ok {
		return false // IP not registered
	}

	userID := userIDVal.(string)
	userVal, ok := users.Load(userID)
	if !ok {
		return false // User not found
	}

	user := userVal.(*User)

	// Check if user is active and not expired
	if !user.IsActive || time.Now().After(user.ExpiresAt) {
		return false
	}

	// Update usage stats
	user.UsageCount++
	user.LastUsed = time.Now()
	users.Store(userID, user)

	return true
}

// Get user by IP (fast lookup)
func getUserByIP(ip string) *User {
	userIDVal, ok := ipToUser.Load(ip)
	if !ok {
		return nil
	}

	userID := userIDVal.(string)
	return getUserByID(userID)
}

// Get user by ID
func getUserByID(id string) *User {
	if val, ok := users.Load(id); ok {
		return val.(*User)
	}
	return nil
}

// Create new user
func createUser(name, description string, maxIPs, validDays int) (*User, error) {
	id, err := generateUserID()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &User{
		ID:          id,
		Name:        name,
		IPs:         []string{},
		MaxIPs:      maxIPs,
		Description: description,
		CreatedAt:   now,
		ExpiresAt:   now.AddDate(0, 0, validDays),
		IsActive:    true,
		UsageCount:  0,
		LastUsed:    time.Time{},
	}

	users.Store(id, user)
	logger.Info("user created", "id", id, "name", name, "max_ips", maxIPs, "expires", user.ExpiresAt)
	return user, nil
}

// Update user expiration
func extendUserExpiration(userID string, days int) error {
	user := getUserByID(userID)
	if user == nil {
		return errors.New("user not found")
	}

	user.ExpiresAt = user.ExpiresAt.AddDate(0, 0, days)
	users.Store(userID, user)
	logger.Info("user expiration extended", "id", userID, "new_expiry", user.ExpiresAt)
	return nil
}

// Deactivate user
func deactivateUser(userID string) error {
	user := getUserByID(userID)
	if user == nil {
		return errors.New("user not found")
	}

	user.IsActive = false
	users.Store(userID, user)
	logger.Info("user deactivated", "id", userID)
	return nil
}

// Delete user
func deleteUser(userID string) error {
	user := getUserByID(userID)
	if user == nil {
		return errors.New("user not found")
	}

	// Remove all IP mappings
	for _, ip := range user.IPs {
		ipToUser.Delete(ip)
	}

	users.Delete(userID)
	logger.Info("user deleted", "id", userID)
	return nil
}

// Add IP to user (FIFO - removes oldest if limit reached)
func addIPToUser(userID, clientIP string) error {
	user := getUserByID(userID)
	if user == nil {
		return errors.New("user not found")
	}

	if !user.IsActive {
		return errors.New("user is inactive")
	}

	if time.Now().After(user.ExpiresAt) {
		return errors.New("user expired")
	}

	// Check if IP already registered
	for _, ip := range user.IPs {
		if ip == clientIP {
			logger.Info("IP already registered for user", "user_id", userID, "ip", clientIP)
			return nil // Already registered, no error
		}
	}

	// If max IPs reached, remove the oldest (FIFO)
	if len(user.IPs) >= user.MaxIPs && user.MaxIPs > 0 {
		oldestIP := user.IPs[0]
		user.IPs = user.IPs[1:] // Remove first element
		ipToUser.Delete(oldestIP)
		logger.Info("removed oldest IP (FIFO)", "user_id", userID, "old_ip", oldestIP)
	}

	// Add new IP
	user.IPs = append(user.IPs, clientIP)
	ipToUser.Store(clientIP, userID)
	users.Store(userID, user)

	logger.Info("IP added to user", "user_id", userID, "ip", clientIP, "total_ips", len(user.IPs))
	return nil
}


// Background task to check and deactivate expired users
func startExpirationChecker(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			checkExpiredUsers()
		}
	}
}

func checkExpiredUsers() {
	now := time.Now()
	expiredCount := 0

	users.Range(func(key, value interface{}) bool {
		user := value.(*User)
		if user.IsActive && now.After(user.ExpiresAt) {
			user.IsActive = false
			users.Store(key, user)
			expiredCount++
			logger.Info("user expired and deactivated", "id", user.ID, "name", user.Name, "expired_at", user.ExpiresAt)
		}
		return true
	})

	if expiredCount > 0 {
		logger.Info("expired users check completed", "deactivated_count", expiredCount)
	}
}

func createSession(username string) (string, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return "", err
	}

	session := &Session{
		Username:  username,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	webSessions.Store(sessionID, session)
	logger.Info("session created", "username", username, "session_id", sessionID[:16]+"...")

	return sessionID, nil
}

func validateSession(sessionID string) (*Session, bool) {
	val, exists := webSessions.Load(sessionID)
	if !exists {
		return nil, false
	}

	session := val.(*Session)
	session.mu.RLock()
	defer session.mu.RUnlock()

	if time.Now().After(session.ExpiresAt) {
		webSessions.Delete(sessionID)
		return nil, false
	}

	return session, true
}

func deleteSession(sessionID string) {
	webSessions.Delete(sessionID)
	logger.Info("session deleted", "session_id", sessionID[:16]+"...")
}

func cleanExpiredSessions() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		webSessions.Range(func(key, value interface{}) bool {
			session := value.(*Session)
			session.mu.RLock()
			expired := now.After(session.ExpiresAt)
			session.mu.RUnlock()

			if expired {
				webSessions.Delete(key)
				logger.Debug("removed expired session", "session_id", key.(string)[:16]+"...")
			}
			return true
		})
	}
}

func checkWebPanelAuth(username, password string) bool {
	cfg := getConfig()

	if !cfg.WebPanelEnabled {
		return false
	}

	if cfg.WebPanelUsername == "" || cfg.WebPanelPassword == "" {
		return false
	}

	hashedPassword := hashPassword(password)
	return username == cfg.WebPanelUsername && hashedPassword == cfg.WebPanelPassword
}

// Domain matcher with wildcard support: "*.example.com"
func matches(host, pattern string) bool {
	h := strings.ToLower(trimDot(host))
	p := strings.ToLower(trimDot(pattern))
	if p == "" {
		return false
	}
	if strings.HasPrefix(p, "*.") {
		suf := p[1:] // ".example.com"
		// require suffix match and at least as many labels as the pattern
		return strings.HasSuffix(h, suf) && countDots(h) >= countDots(p)
	}
	return h == p
}

func findValueByPattern(m map[string]string, host string) (string, bool) {
	for k, v := range m {
		if matches(host, k) {
			return v, true
		}
	}
	return "", false
}

// ======================== Metrics ========================

func (m *Metrics) IncDOHQueries() {
	atomic.AddUint64(&m.dohQueries, 1)
}

func (m *Metrics) IncDOTQueries() {
	atomic.AddUint64(&m.dotQueries, 1)
}

func (m *Metrics) IncSNIConnections() {
	atomic.AddUint64(&m.sniConnections, 1)
}

func (m *Metrics) IncCacheHits() {
	atomic.AddUint64(&m.cacheHits, 1)
}

func (m *Metrics) IncCacheMisses() {
	atomic.AddUint64(&m.cacheMisses, 1)
}

func (m *Metrics) IncErrors() {
	atomic.AddUint64(&m.errors, 1)
}

func (m *Metrics) GetStats() map[string]uint64 {
	return map[string]uint64{
		"doh_queries":     atomic.LoadUint64(&m.dohQueries),
		"dot_queries":     atomic.LoadUint64(&m.dotQueries),
		"sni_connections": atomic.LoadUint64(&m.sniConnections),
		"cache_hits":      atomic.LoadUint64(&m.cacheHits),
		"cache_misses":    atomic.LoadUint64(&m.cacheMisses),
		"errors":          atomic.LoadUint64(&m.errors),
	}
}

// ======================== DNS Cache ========================

func getCacheKey(query []byte) string {
	return base64.StdEncoding.EncodeToString(query)
}

func getCachedResponse(query []byte) ([]byte, bool) {
	key := getCacheKey(query)
	val, exists := dnsCache.Load(key)
	if !exists {
		metrics.IncCacheMisses()
		return nil, false
	}

	entry := val.(*CacheEntry)
	entry.mu.RLock()
	defer entry.mu.RUnlock()

	if time.Now().After(entry.ExpiresAt) {
		dnsCache.Delete(key)
		metrics.IncCacheMisses()
		return nil, false
	}

	metrics.IncCacheHits()
	logger.Debug("cache hit", "key", key[:20]+"...")
	return entry.Response, true
}

func setCachedResponse(query, response []byte) {
	cfg := getConfig()
	if cfg.CacheTTL <= 0 {
		return
	}

	key := getCacheKey(query)
	entry := &CacheEntry{
		Response:  response,
		ExpiresAt: time.Now().Add(time.Duration(cfg.CacheTTL) * time.Second),
	}

	dnsCache.Store(key, entry)
	logger.Debug("cached response", "key", key[:20]+"...", "ttl", cfg.CacheTTL)
}

func cleanExpiredCache() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		dnsCache.Range(func(key, value interface{}) bool {
			entry := value.(*CacheEntry)
			entry.mu.RLock()
			expired := now.After(entry.ExpiresAt)
			entry.mu.RUnlock()

			if expired {
				dnsCache.Delete(key)
				logger.Debug("removed expired cache entry", "key", key.(string)[:20]+"...")
			}
			return true
		})
	}
}

// ======================== DNS Handling ========================

func buildLocalDNSResponse(req *dns.Msg, ipStr string) ([]byte, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	resp := new(dns.Msg)
	resp.SetReply(req)
	resp.RecursionAvailable = true
	resp.Compress = true

	q := req.Question[0]
	name := q.Name

	// Answer only if the type matches the IP version.
	switch q.Qtype {
	case dns.TypeA:
		if ip4 := ip.To4(); ip4 != nil {
			resp.Answer = append(resp.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:   name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    defaultTTL,
				},
				A: ip4,
			})
		}
	case dns.TypeAAAA:
		if ip16 := ip.To16(); ip16 != nil && ip.To4() == nil {
			resp.Answer = append(resp.Answer, &dns.AAAA{
				Hdr: dns.RR_Header{
					Name:   name,
					Rrtype: dns.TypeAAAA,
					Class:  dns.ClassINET,
					Ttl:    defaultTTL,
				},
				AAAA: ip16,
			})
		}
	case dns.TypeANY:
		// Return whichever record matches the IP family
		if ip4 := ip.To4(); ip4 != nil {
			resp.Answer = append(resp.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:   name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    defaultTTL,
				},
				A: ip4,
			})
		} else if ip16 := ip.To16(); ip16 != nil {
			resp.Answer = append(resp.Answer, &dns.AAAA{
				Hdr: dns.RR_Header{
					Name:   name,
					Rrtype: dns.TypeAAAA,
					Class:  dns.ClassINET,
					Ttl:    defaultTTL,
				},
				AAAA: ip16,
			})
		}
	default:
		// NOERROR / NODATA for other types
	}

	return resp.Pack()
}

func processDNSQuery(query []byte) ([]byte, error) {
	var req dns.Msg
	if err := req.Unpack(query); err != nil {
		logger.Warn("failed to unpack DNS query", "error", err)
		return nil, err
	}
	if len(req.Question) == 0 {
		return nil, errors.New("no DNS question")
	}

	qName := trimDot(req.Question[0].Name)
	qType := req.Question[0].Qtype

	logger.Debug("processing DNS query", "domain", qName, "type", dns.TypeToString[qType])

	// Check if domain is blocked
	if isDomainBlocked(qName) {
		logger.Info("blocked domain query", "domain", qName)
		metrics.IncErrors()
		return buildBlockedResponse(&req)
	}

	// Check cache first
	if cached, found := getCachedResponse(query); found {
		logger.Debug("returning cached response", "domain", qName)
		return cached, nil
	}

	// Check local domains
	cfg := getConfig()
	if ip, ok := findValueByPattern(cfg.Domains, qName); ok {
		logger.Debug("local domain match", "domain", qName, "ip", ip)
		resp, err := buildLocalDNSResponse(&req, ip)
		if err == nil {
			setCachedResponse(query, resp)
		}
		return resp, err
	}

	// Forward to upstream DoH with failover
	upstreams := dohUpstream.Load().([]string)
	var lastErr error

	for _, upstream := range upstreams {
		resp, err := queryUpstreamDoH(upstream, query)
		if err == nil {
			setCachedResponse(query, resp)
			logger.Debug("upstream query success", "domain", qName, "upstream", upstream)
			return resp, nil
		}
		logger.Warn("upstream query failed", "domain", qName, "upstream", upstream, "error", err)
		lastErr = err
	}

	metrics.IncErrors()
	return nil, fmt.Errorf("all upstream servers failed, last error: %w", lastErr)
}

func queryUpstreamDoH(upstream string, query []byte) ([]byte, error) {
	httpReq, err := http.NewRequest("POST", upstream, bytes.NewReader(query))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/dns-message")
	httpReq.Header.Set("Accept", "application/dns-message")
	httpReq.Header.Set("User-Agent", "smartSNI/2.0")

	resp, err := dohClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slurp, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("upstream status %d: %s", resp.StatusCode, string(slurp))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func buildBlockedResponse(req *dns.Msg) ([]byte, error) {
	resp := new(dns.Msg)
	resp.SetReply(req)
	resp.RecursionAvailable = true
	resp.Rcode = dns.RcodeRefused
	return resp.Pack()
}

// ======================== DoT Server ========================

func handleDoTConnection(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("panic in DoT handler", "error", r, "stack", string(debug.Stack()))
			metrics.IncErrors()
		}
		conn.Close()
	}()

	metrics.IncDOTQueries()
	clientAddr := conn.RemoteAddr().String()
	clientIP, _, _ := net.SplitHostPort(clientAddr)

	logger.Debug("DoT connection", "client", clientAddr)

	// Check user-based authorization
	if !isUserAuthorized(clientIP) {
		logger.Warn("DoT user not authorized", "client", clientIP)
		metrics.IncErrors()
		return
	}

	if !limiter.Allow() {
		logger.Warn("DoT global rate limit exceeded", "client", clientAddr)
		metrics.IncErrors()
		return
	}

	// Set read deadline
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		logger.Error("DoT set read deadline failed", "error", err)
		return
	}

	// DoT framing: 2-byte length + DNS payload (RFC 7858 uses TCP DNS framing)
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		logger.Debug("DoT read length failed", "error", err, "client", clientAddr)
		metrics.IncErrors()
		return
	}

	dnsLen := binary.BigEndian.Uint16(header)
	// Basic sanity limit to avoid huge allocations
	if dnsLen == 0 || dnsLen > 8192 {
		logger.Warn("DoT invalid length", "length", dnsLen, "client", clientAddr)
		metrics.IncErrors()
		return
	}

	buf := make([]byte, int(dnsLen))
	if _, err := io.ReadFull(conn, buf); err != nil {
		logger.Debug("DoT read body failed", "error", err, "client", clientAddr)
		metrics.IncErrors()
		return
	}

	resp, err := processDNSQuery(buf)
	if err != nil {
		logger.Warn("DoT query processing failed", "error", err, "client", clientAddr)
		metrics.IncErrors()
		return
	}

	// Set write deadline
	if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		logger.Error("DoT set write deadline failed", "error", err)
		return
	}

	outLen := make([]byte, 2)
	binary.BigEndian.PutUint16(outLen, uint16(len(resp)))
	if _, err := conn.Write(outLen); err != nil {
		logger.Debug("DoT write length failed", "error", err, "client", clientAddr)
		metrics.IncErrors()
		return
	}
	if _, err := conn.Write(resp); err != nil {
		logger.Debug("DoT write body failed", "error", err, "client", clientAddr)
		metrics.IncErrors()
		return
	}

	logger.Debug("DoT query completed successfully", "client", clientAddr)
}

func startDoTServer(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	cfg := getConfig()
	certDir := filepath.Join("/etc/letsencrypt/live", cfg.Host)
	cer, err := tls.LoadX509KeyPair(
		filepath.Join(certDir, "fullchain.pem"),
		filepath.Join(certDir, "privkey.pem"),
	)
	if err != nil {
		logger.Warn("DoT: SSL certificate not found, DoT server disabled", "error", err, "cert_dir", certDir)
		logger.Warn("DoT: to enable DoT, obtain SSL certificate with: certbot --nginx -d <domain>")
		return
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cer},
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
		PreferServerCipherSuites: true,
	}

	ln, err := tls.Listen("tcp", ":853", tlsCfg)
	if err != nil {
		logger.Error("DoT: failed to listen", "error", err)
		log.Fatal("DoT: listen:", err)
	}
	logger.Info("DoT server started", "port", 853)

	go func() {
		<-ctx.Done()
		logger.Info("DoT server shutting down")
		_ = ln.Close()
	}()

	for {
		c, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Warn("DoT accept error", "error", err)
			continue
		}
		go handleDoTConnection(c)
	}
}

// ======================== Standard DNS Server (Port 53) ========================

func startDNSServer(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	// Start UDP DNS server
	udpAddr, err := net.ResolveUDPAddr("udp", ":53")
	if err != nil {
		logger.Error("DNS: failed to resolve UDP address", "error", err)
		return
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		logger.Error("DNS: failed to listen on UDP:53", "error", err)
		logger.Warn("DNS: standard DNS server disabled (port 53 unavailable)")
		return
	}
	defer udpConn.Close()

	// Start TCP DNS server
	tcpAddr, err := net.ResolveTCPAddr("tcp", ":53")
	if err != nil {
		logger.Error("DNS: failed to resolve TCP address", "error", err)
		return
	}

	tcpListener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		logger.Error("DNS: failed to listen on TCP:53", "error", err)
		return
	}
	defer tcpListener.Close()

	logger.Info("DNS server started", "port", 53, "protocols", "UDP/TCP")

	// Handle shutdown
	go func() {
		<-ctx.Done()
		logger.Info("DNS server shutting down")
		udpConn.Close()
		tcpListener.Close()
	}()

	// Start TCP handler in goroutine
	go func() {
		for {
			conn, err := tcpListener.Accept()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				logger.Warn("DNS TCP accept error", "error", err)
				continue
			}
			go handleDNSTCP(conn)
		}
	}()

	// Handle UDP requests
	buf := make([]byte, 512)
	for {
		n, addr, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		go handleDNSUDP(udpConn, addr, buf[:n])
	}
}

func handleDNSUDP(conn *net.UDPConn, addr *net.UDPAddr, query []byte) {
	response, err := processDNSQuery(query)
	if err != nil {
		logger.Debug("DNS UDP query failed", "error", err, "client", addr.String())
		return
	}
	_, _ = conn.WriteToUDP(response, addr)
}

func handleDNSTCP(conn net.Conn) {
	defer conn.Close()

	// Read DNS message length (2 bytes)
	lenBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return
	}
	msgLen := binary.BigEndian.Uint16(lenBuf)

	// Read DNS query
	query := make([]byte, msgLen)
	if _, err := io.ReadFull(conn, query); err != nil {
		return
	}

	response, err := processDNSQuery(query)
	if err != nil {
		return
	}

	// Write response length + response
	respLen := make([]byte, 2)
	binary.BigEndian.PutUint16(respLen, uint16(len(response)))
	conn.Write(respLen)
	conn.Write(response)
}

// ======================== TLS SNI Peek ========================

type readOnlyConn struct{ r io.Reader }

func (c readOnlyConn) Read(p []byte) (int, error)       { return c.r.Read(p) }
func (c readOnlyConn) Write(_ []byte) (int, error)      { return 0, io.ErrClosedPipe }
func (c readOnlyConn) Close() error                     { return nil }
func (c readOnlyConn) LocalAddr() net.Addr              { return nil }
func (c readOnlyConn) RemoteAddr() net.Addr             { return nil }
func (c readOnlyConn) SetDeadline(time.Time) error      { return nil }
func (c readOnlyConn) SetReadDeadline(time.Time) error  { return nil }
func (c readOnlyConn) SetWriteDeadline(time.Time) error { return nil }

// Perform a TLS handshake only to capture ClientHello (SNI), then abort
func readClientHello(reader io.Reader) (*tls.ClientHelloInfo, error) {
	helloCh := make(chan *tls.ClientHelloInfo, 1)

	cfg := &tls.Config{
		GetConfigForClient: func(chi *tls.ClientHelloInfo) (*tls.Config, error) {
			select {
			case helloCh <- chi:
			default:
			}
			// Returning nil causes handshake to fail fast; we only need the Hello
			return nil, nil
		},
	}

	t := tls.Server(readOnlyConn{r: reader}, cfg)
	_ = t.Handshake() // expected to error; we just want ClientHello

	select {
	case h := <-helloCh:
		return h, nil
	default:
		return nil, errors.New("failed to capture ClientHello")
	}
}

func peekClientHello(reader io.Reader) (*tls.ClientHelloInfo, io.Reader, error) {
	peekBuf := new(bytes.Buffer)
	hello, err := readClientHello(io.TeeReader(reader, peekBuf))
	if err != nil {
		return nil, nil, err
	}
	return hello, peekBuf, nil
}

// ======================== TCP Proxy (SNI) ========================

func closeWrite(c net.Conn) {
	if tc, ok := c.(*net.TCPConn); ok {
		_ = tc.CloseWrite()
	}
}

func copyWithPool(dst, src net.Conn) {
	buf := BufferPool.Get().([]byte)
	defer BufferPool.Put(buf)
	_, _ = io.CopyBuffer(dst, src, buf)
	closeWrite(dst)
}

func handleConnection(clientConn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("panic in SNI handler", "error", r, "stack", string(debug.Stack()))
			metrics.IncErrors()
		}
		clientConn.Close()
	}()

	metrics.IncSNIConnections()
	clientAddr := clientConn.RemoteAddr().String()

	logger.Debug("SNI connection", "client", clientAddr)

	// Deadline only for initial ClientHello capture
	_ = clientConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	clientHello, clientHelloBytes, err := peekClientHello(clientConn)
	if err != nil {
		logger.Debug("SNI peek failed", "error", err, "client", clientAddr)
		metrics.IncErrors()
		return
	}
	_ = clientConn.SetReadDeadline(time.Time{}) // clear deadline

	sni := strings.TrimSpace(strings.ToLower(clientHello.ServerName))
	if sni == "" {
		logger.Warn("SNI missing from ClientHello", "client", clientAddr)
		metrics.IncErrors()
		// Meaningful HTTP error for non-TLS/empty SNI traffic
		resp := "HTTP/1.1 421 Misdirected Request\r\n" +
			"Content-Type: text/plain; charset=utf-8\r\n" +
			"Connection: close\r\n" +
			"Content-Length: 12\r\n\r\nSNI required"
		_, _ = clientConn.Write([]byte(resp))
		return
	}

	logger.Debug("SNI detected", "sni", sni, "client", clientAddr)

	cfg := getConfig()
	target := sni
	if target == strings.ToLower(cfg.Host) {
		target = "127.0.0.1:8443"
		logger.Debug("routing to local HTTPS", "sni", sni)
	} else {
		target = net.JoinHostPort(target, "443")
	}

	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	backendConn, err := dialer.Dial("tcp", target)
	if err != nil {
		logger.Warn("backend dial failed", "target", target, "error", err, "client", clientAddr)
		metrics.IncErrors()
		return
	}
	defer backendConn.Close()

	// Replay the captured ClientHello to the backend first
	if _, err := io.Copy(backendConn, clientHelloBytes); err != nil {
		logger.Debug("failed to write ClientHello", "error", err, "client", clientAddr)
		metrics.IncErrors()
		return
	}

	logger.Debug("proxying connection", "sni", sni, "target", target, "client", clientAddr)

	// Bidirectional relay
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		copyWithPool(clientConn, backendConn) // backend -> client
	}()
	go func() {
		defer wg.Done()
		copyWithPool(backendConn, clientConn) // client -> backend
	}()

	wg.Wait()
	logger.Debug("SNI connection closed", "sni", sni, "client", clientAddr)
}

func serveSniProxy(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	cfg := getConfig()
	sniPort := cfg.SNIPort
	if sniPort == 0 {
		sniPort = 443
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", sniPort))
	if err != nil {
		logger.Error("SNI: failed to listen", "error", err)
		logger.Warn("SNI proxy disabled due to port conflict")
		return
	}
	logger.Info("SNI proxy started", "port", sniPort)

	go func() {
		<-ctx.Done()
		logger.Info("SNI proxy shutting down")
		_ = ln.Close()
	}()

	for {
		c, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Warn("SNI accept error", "error", err)
			continue
		}
		go handleConnection(c)
	}
}

// ======================== DoH Server (fasthttp) ========================

func handleDoHRequest(ctx *fasthttp.RequestCtx) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("panic in DoH handler", "error", r, "stack", string(debug.Stack()))
			metrics.IncErrors()
			ctx.Error("Internal server error", fasthttp.StatusInternalServerError)
		}
	}()

	metrics.IncDOHQueries()
	clientIP := getClientIP(ctx)

	logger.Debug("DoH request", "client", clientIP, "method", string(ctx.Method()))

	// Check user-based authorization
	if !isUserAuthorized(clientIP) {
		logger.Warn("DoH user not authorized", "client", clientIP)
		metrics.IncErrors()
		ctx.Error("Access denied - Please register first", fasthttp.StatusForbidden)
		return
	}

	// Check authentication
	if !checkAuth(ctx) {
		logger.Warn("DoH authentication failed", "client", clientIP)
		metrics.IncErrors()
		ctx.Error("Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	// Global rate limit
	if !limiter.Allow() {
		logger.Warn("DoH global rate limit exceeded", "client", clientIP)
		metrics.IncErrors()
		ctx.Error("Rate limit exceeded", fasthttp.StatusTooManyRequests)
		return
	}

	// Per-IP rate limit
	ipLimiter := getIPLimiter(clientIP)
	if !ipLimiter.Allow() {
		logger.Warn("DoH per-IP rate limit exceeded", "client", clientIP)
		metrics.IncErrors()
		ctx.Error("Rate limit exceeded", fasthttp.StatusTooManyRequests)
		return
	}

	var body []byte
	switch string(ctx.Method()) {
	case "GET":
		raw := ctx.QueryArgs().Peek("dns")
		if raw == nil {
			logger.Debug("DoH missing dns parameter", "client", clientIP)
			ctx.Error("Missing 'dns' query parameter", fasthttp.StatusBadRequest)
			return
		}
		decoded, err := base64.RawURLEncoding.DecodeString(string(raw))
		if err != nil {
			logger.Warn("DoH invalid dns parameter", "client", clientIP, "error", err)
			metrics.IncErrors()
			ctx.Error("Invalid 'dns' query parameter", fasthttp.StatusBadRequest)
			return
		}
		body = decoded
	case "POST":
		body = ctx.PostBody()
		if len(body) == 0 {
			logger.Debug("DoH empty request body", "client", clientIP)
			ctx.Error("Empty request body", fasthttp.StatusBadRequest)
			return
		}
	default:
		logger.Debug("DoH invalid method", "client", clientIP, "method", string(ctx.Method()))
		ctx.Error("Only GET and POST methods are allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	// Validate DNS query size
	if len(body) > 4096 {
		logger.Warn("DoH query too large", "client", clientIP, "size", len(body))
		metrics.IncErrors()
		ctx.Error("DNS query too large", fasthttp.StatusRequestEntityTooLarge)
		return
	}

	resp, err := processDNSQuery(body)
	if err != nil {
		logger.Warn("DoH query processing failed", "client", clientIP, "error", err)
		metrics.IncErrors()
		ctx.Error("Failed to process DNS query", fasthttp.StatusBadRequest)
		return
	}

	// Security headers
	ctx.Response.Header.Set("X-Content-Type-Options", "nosniff")
	ctx.Response.Header.Set("X-Frame-Options", "DENY")
	ctx.Response.Header.Set("X-XSS-Protection", "1; mode=block")
	ctx.Response.Header.Set("Referrer-Policy", "no-referrer")

	ctx.SetContentType("application/dns-message")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.Write(resp)

	logger.Debug("DoH query completed", "client", clientIP)
}

func handleHealthCheck(ctx *fasthttp.RequestCtx) {
	health := map[string]interface{}{
		"status": "healthy",
		"uptime": time.Since(startTime).Seconds(),
		"version": "2.0",
	}

	data, _ := json.Marshal(health)
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.Write(data)
}

func handleMetrics(ctx *fasthttp.RequestCtx) {
	cfg := getConfig()
	if !cfg.MetricsEnabled {
		ctx.Error("Metrics disabled", fasthttp.StatusForbidden)
		return
	}

	stats := metrics.GetStats()
	data, _ := json.Marshal(stats)

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.Write(data)
}

func handlePrometheusMetrics(ctx *fasthttp.RequestCtx) {
	cfg := getConfig()
	if !cfg.MetricsEnabled {
		ctx.Error("Metrics disabled", fasthttp.StatusForbidden)
		return
	}

	stats := metrics.GetStats()
	var sb strings.Builder

	sb.WriteString("# HELP smartsni_doh_queries_total Total number of DoH queries\n")
	sb.WriteString("# TYPE smartsni_doh_queries_total counter\n")
	sb.WriteString(fmt.Sprintf("smartsni_doh_queries_total %d\n", stats["doh_queries"]))

	sb.WriteString("# HELP smartsni_dot_queries_total Total number of DoT queries\n")
	sb.WriteString("# TYPE smartsni_dot_queries_total counter\n")
	sb.WriteString(fmt.Sprintf("smartsni_dot_queries_total %d\n", stats["dot_queries"]))

	sb.WriteString("# HELP smartsni_sni_connections_total Total number of SNI connections\n")
	sb.WriteString("# TYPE smartsni_sni_connections_total counter\n")
	sb.WriteString(fmt.Sprintf("smartsni_sni_connections_total %d\n", stats["sni_connections"]))

	sb.WriteString("# HELP smartsni_cache_hits_total Total number of cache hits\n")
	sb.WriteString("# TYPE smartsni_cache_hits_total counter\n")
	sb.WriteString(fmt.Sprintf("smartsni_cache_hits_total %d\n", stats["cache_hits"]))

	sb.WriteString("# HELP smartsni_cache_misses_total Total number of cache misses\n")
	sb.WriteString("# TYPE smartsni_cache_misses_total counter\n")
	sb.WriteString(fmt.Sprintf("smartsni_cache_misses_total %d\n", stats["cache_misses"]))

	sb.WriteString("# HELP smartsni_errors_total Total number of errors\n")
	sb.WriteString("# TYPE smartsni_errors_total counter\n")
	sb.WriteString(fmt.Sprintf("smartsni_errors_total %d\n", stats["errors"]))

	ctx.SetContentType("text/plain; version=0.0.4")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.WriteString(sb.String())
}

func handleConfigReload(ctx *fasthttp.RequestCtx) {
	// Simple auth check
	if !checkAuth(ctx) {
		ctx.Error("Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	if err := reloadConfig("config.json"); err != nil {
		logger.Error("failed to reload config", "error", err)
		ctx.Error(fmt.Sprintf("Failed to reload: %v", err), fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.WriteString(`{"status":"reloaded"}`)
}

var startTime = time.Now()

// ======================== Web Panel Handlers ========================

func serveWebPanel(ctx *fasthttp.RequestCtx) {
	htmlContent, err := os.ReadFile("webpanel.html")
	if err != nil {
		ctx.Error("Web panel not found", fasthttp.StatusNotFound)
		return
	}

	ctx.SetContentType("text/html; charset=utf-8")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.Write(htmlContent)
}

func handlePanelLogin(ctx *fasthttp.RequestCtx) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.Error(`{"error":"Invalid request"}`, fasthttp.StatusBadRequest)
		return
	}

	if !checkWebPanelAuth(req.Username, req.Password) {
		logger.Warn("web panel login failed", "username", req.Username, "ip", ctx.RemoteIP().String())
		ctx.SetStatusCode(fasthttp.StatusUnauthorized)
		_, _ = ctx.WriteString(`{"error":"Invalid credentials"}`)
		return
	}

	sessionID, err := createSession(req.Username)
	if err != nil {
		ctx.Error(`{"error":"Failed to create session"}`, fasthttp.StatusInternalServerError)
		return
	}

	logger.Info("web panel login successful", "username", req.Username)

	resp, _ := json.Marshal(map[string]string{
		"session_id": sessionID,
		"username":   req.Username,
	})

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.Write(resp)
}

func handlePanelLogout(ctx *fasthttp.RequestCtx) {
	sessionID := string(ctx.Request.Header.Peek("X-Session-ID"))
	if sessionID != "" {
		deleteSession(sessionID)
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.WriteString(`{"status":"logged_out"}`)
}

func handlePanelValidate(ctx *fasthttp.RequestCtx) {
	sessionID := string(ctx.Request.Header.Peek("X-Session-ID"))
	session, valid := validateSession(sessionID)

	if !valid {
		ctx.SetStatusCode(fasthttp.StatusUnauthorized)
		_, _ = ctx.WriteString(`{"error":"Invalid session"}`)
		return
	}

	resp, _ := json.Marshal(map[string]string{
		"username": session.Username,
	})

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.Write(resp)
}

func requirePanelAuth(ctx *fasthttp.RequestCtx) bool {
	sessionID := string(ctx.Request.Header.Peek("X-Session-ID"))
	_, valid := validateSession(sessionID)

	if !valid {
		ctx.SetStatusCode(fasthttp.StatusUnauthorized)
		_, _ = ctx.WriteString(`{"error":"Unauthorized"}`)
		return false
	}

	return true
}

func handlePanelMetrics(ctx *fasthttp.RequestCtx) {
	if !requirePanelAuth(ctx) {
		return
	}

	stats := metrics.GetStats()
	data, _ := json.Marshal(stats)

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.Write(data)
}

func handlePanelHealth(ctx *fasthttp.RequestCtx) {
	if !requirePanelAuth(ctx) {
		return
	}

	health := map[string]interface{}{
		"status":  "healthy",
		"uptime":  time.Since(startTime).Seconds(),
		"version": "2.0",
	}

	data, _ := json.Marshal(health)
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.Write(data)
}

func handlePanelDomains(ctx *fasthttp.RequestCtx) {
	if !requirePanelAuth(ctx) {
		return
	}

	cfg := getConfig()
	resp, _ := json.Marshal(map[string]interface{}{
		"domains": cfg.Domains,
		"host":    cfg.Host,
	})

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.Write(resp)
}

func handlePanelAddDomain(ctx *fasthttp.RequestCtx) {
	if !requirePanelAuth(ctx) {
		return
	}

	var req struct {
		Domain string `json:"domain"`
	}

	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.Error(`{"error":"Invalid request"}`, fasthttp.StatusBadRequest)
		return
	}

	// Read current config
	cfg, err := LoadConfig("config.json")
	if err != nil {
		logger.Error("failed to load config for add domain", "error", err)
		ctx.Error(fmt.Sprintf(`{"error":"Failed to load config: %s"}`, err.Error()), fasthttp.StatusInternalServerError)
		return
	}

	// Get server IP from existing domains or use first domain's IP
	var serverIP string
	for _, ip := range cfg.Domains {
		if net.ParseIP(ip) != nil {
			serverIP = ip
			break
		}
	}

	if serverIP == "" {
		ctx.Error(`{"error":"No server IP found in config. Please add at least one domain first via install.sh"}`, fasthttp.StatusBadRequest)
		return
	}

	// Clean domain
	domain := strings.TrimSpace(req.Domain)
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimSuffix(domain, "/")

	if domain == "" {
		ctx.Error(`{"error":"Domain cannot be empty"}`, fasthttp.StatusBadRequest)
		return
	}

	// Add both exact and wildcard patterns
	if strings.HasPrefix(domain, "*.") || strings.Contains(domain, "*") {
		// Already has wildcard, add as-is
		cfg.Domains[domain] = serverIP
	} else {
		// Add both exact and wildcard
		cfg.Domains[domain] = serverIP
		cfg.Domains["*."+domain] = serverIP
	}

	// Save config
	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile("config.json", data, 0644); err != nil {
		ctx.Error(`{"error":"Failed to save config"}`, fasthttp.StatusInternalServerError)
		return
	}

	// Reload config
	if err := reloadConfig("config.json"); err != nil {
		logger.Error("failed to reload after adding domain", "error", err)
	}

	logger.Info("domain added via web panel", "domain", domain, "wildcard", "*."+domain, "ip", serverIP)

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.WriteString(`{"status":"added"}`)
}

func handlePanelRemoveDomain(ctx *fasthttp.RequestCtx) {
	if !requirePanelAuth(ctx) {
		return
	}

	var req struct {
		Domain string `json:"domain"`
	}

	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.Error(`{"error":"Invalid request"}`, fasthttp.StatusBadRequest)
		return
	}

	// Read current config
	cfg, err := LoadConfig("config.json")
	if err != nil {
		ctx.Error(`{"error":"Failed to load config"}`, fasthttp.StatusInternalServerError)
		return
	}

	// Remove domain
	delete(cfg.Domains, req.Domain)

	// Save config
	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile("config.json", data, 0644); err != nil {
		ctx.Error(`{"error":"Failed to save config"}`, fasthttp.StatusInternalServerError)
		return
	}

	// Reload config
	if err := reloadConfig("config.json"); err != nil {
		logger.Error("failed to reload after removing domain", "error", err)
	}

	logger.Info("domain removed via web panel", "domain", req.Domain)

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.WriteString(`{"status":"removed"}`)
}

func handlePanelReload(ctx *fasthttp.RequestCtx) {
	if !requirePanelAuth(ctx) {
		return
	}

	if err := reloadConfig("config.json"); err != nil {
		logger.Error("failed to reload config via panel", "error", err)
		ctx.Error(`{"error":"Failed to reload"}`, fasthttp.StatusInternalServerError)
		return
	}

	logger.Info("config reloaded via web panel")

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.WriteString(`{"status":"reloaded"}`)
}

// ======================== User Management API Handlers ========================

func handlePanelUsers(ctx *fasthttp.RequestCtx) {
	if !requirePanelAuth(ctx) {
		return
	}

	var userList []User
	users.Range(func(key, value interface{}) bool {
		user := value.(*User)
		userList = append(userList, *user)
		return true
	})

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(map[string]interface{}{
		"users": userList,
		"count": len(userList),
	})
}

func handlePanelCreateUser(ctx *fasthttp.RequestCtx) {
	if !requirePanelAuth(ctx) {
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		MaxIPs      int    `json:"max_ips"`
		ValidDays   int    `json:"valid_days"`
	}

	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.Error(`{"error":"Invalid JSON"}`, fasthttp.StatusBadRequest)
		return
	}

	if req.Name == "" || req.MaxIPs <= 0 || req.ValidDays <= 0 {
		ctx.Error(`{"error":"Missing required fields"}`, fasthttp.StatusBadRequest)
		return
	}

	user, err := createUser(req.Name, req.Description, req.MaxIPs, req.ValidDays)
	if err != nil {
		logger.Error("failed to create user", "error", err)
		ctx.Error(`{"error":"Failed to create user"}`, fasthttp.StatusInternalServerError)
		return
	}

	cfg := getConfig()
	registerURL := fmt.Sprintf("http://%s:%d/register?token=%s", cfg.Host, cfg.WebPanelPort, user.ID)

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(map[string]interface{}{
		"success":      true,
		"user":         user,
		"register_url": registerURL,
	})
}

func handlePanelExtendUser(ctx *fasthttp.RequestCtx) {
	if !requirePanelAuth(ctx) {
		return
	}

	var req struct {
		UserID string `json:"user_id"`
		Days   int    `json:"days"`
	}

	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.Error(`{"error":"Invalid JSON"}`, fasthttp.StatusBadRequest)
		return
	}

	if err := extendUserExpiration(req.UserID, req.Days); err != nil {
		ctx.Error(fmt.Sprintf(`{"error":"%s"}`, err.Error()), fasthttp.StatusNotFound)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.WriteString(`{"success":true}`)
}

func handlePanelDeactivateUser(ctx *fasthttp.RequestCtx) {
	if !requirePanelAuth(ctx) {
		return
	}

	var req struct {
		UserID string `json:"user_id"`
	}

	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.Error(`{"error":"Invalid JSON"}`, fasthttp.StatusBadRequest)
		return
	}

	if err := deactivateUser(req.UserID); err != nil {
		ctx.Error(fmt.Sprintf(`{"error":"%s"}`, err.Error()), fasthttp.StatusNotFound)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.WriteString(`{"success":true}`)
}

func handlePanelDeleteUser(ctx *fasthttp.RequestCtx) {
	if !requirePanelAuth(ctx) {
		return
	}

	var req struct {
		UserID string `json:"user_id"`
	}

	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.Error(`{"error":"Invalid JSON"}`, fasthttp.StatusBadRequest)
		return
	}

	if err := deleteUser(req.UserID); err != nil {
		ctx.Error(fmt.Sprintf(`{"error":"%s"}`, err.Error()), fasthttp.StatusNotFound)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.WriteString(`{"success":true}`)
}

func handlePanelChangePassword(ctx *fasthttp.RequestCtx) {
	if !requirePanelAuth(ctx) {
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.Error(`{"error":"Invalid JSON"}`, fasthttp.StatusBadRequest)
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		ctx.Error(`{"error":"Current password and new password are required"}`, fasthttp.StatusBadRequest)
		return
	}

	cfg := getConfig()

	// Verify current password
	currentHash := fmt.Sprintf("%x", sha256.Sum256([]byte(req.CurrentPassword)))
	if currentHash != cfg.WebPanelPassword {
		ctx.Error(`{"error":"Current password is incorrect"}`, fasthttp.StatusUnauthorized)
		return
	}

	// Hash new password
	newHash := fmt.Sprintf("%x", sha256.Sum256([]byte(req.NewPassword)))

	// Update config
	cfg.WebPanelPassword = newHash
	config.Store(cfg)

	// Save to file
	if err := SaveConfig("config.json", cfg); err != nil {
		logger.Error("failed to save config after password change", "error", err)
		ctx.Error(`{"error":"Failed to save new password"}`, fasthttp.StatusInternalServerError)
		return
	}

	logger.Info("web panel password changed", "username", cfg.WebPanelUsername)

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.WriteString(`{"success":true,"message":"Password changed successfully"}`)
}

func handlePanelChangeUsername(ctx *fasthttp.RequestCtx) {
	if !requirePanelAuth(ctx) {
		return
	}

	var req struct {
		Password    string `json:"password"`
		NewUsername string `json:"new_username"`
	}

	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.Error(`{"error":"Invalid JSON"}`, fasthttp.StatusBadRequest)
		return
	}

	if req.Password == "" || req.NewUsername == "" {
		ctx.Error(`{"error":"Password and new username are required"}`, fasthttp.StatusBadRequest)
		return
	}

	cfg := getConfig()

	// Verify password
	passwordHash := fmt.Sprintf("%x", sha256.Sum256([]byte(req.Password)))
	if passwordHash != cfg.WebPanelPassword {
		ctx.Error(`{"error":"Password is incorrect"}`, fasthttp.StatusUnauthorized)
		return
	}

	oldUsername := cfg.WebPanelUsername

	// Update config
	cfg.WebPanelUsername = req.NewUsername
	config.Store(cfg)

	// Save to file
	if err := SaveConfig("config.json", cfg); err != nil {
		logger.Error("failed to save config after username change", "error", err)
		ctx.Error(`{"error":"Failed to save new username"}`, fasthttp.StatusInternalServerError)
		return
	}

	logger.Info("web panel username changed", "old_username", oldUsername, "new_username", req.NewUsername)

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.WriteString(`{"success":true,"message":"Username changed successfully"}`)
}

// ======================== Registration Handlers ========================

func serveRegisterPage(ctx *fasthttp.RequestCtx) {
	token := string(ctx.QueryArgs().Peek("token"))
	if token == "" {
		ctx.Error("Missing token", fasthttp.StatusBadRequest)
		return
	}

	// Validate user exists and is valid
	user := getUserByID(token)
	if user == nil {
		ctx.Error("Invalid token", fasthttp.StatusNotFound)
		return
	}

	if !user.IsActive || time.Now().After(user.ExpiresAt) {
		ctx.Error("User expired or inactive", fasthttp.StatusForbidden)
		return
	}

	// Get client IP
	clientIP := getClientIP(ctx)

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Register for DNS Access</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            min-height: 100vh;
            display: flex;
            justify-content: center;
            align-items: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 20px;
            padding: 40px;
            max-width: 500px;
            width: 100%%;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
        }
        h1 {
            color: #667eea;
            margin-bottom: 10px;
            font-size: 28px;
        }
        .subtitle {
            color: #666;
            margin-bottom: 30px;
            font-size: 14px;
        }
        .form-group {
            margin-bottom: 20px;
        }
        label {
            display: block;
            margin-bottom: 8px;
            color: #333;
            font-weight: 500;
        }
        input, textarea {
            width: 100%%;
            padding: 12px;
            border: 2px solid #e0e0e0;
            border-radius: 8px;
            font-size: 14px;
            transition: border-color 0.3s;
        }
        input:focus, textarea:focus {
            outline: none;
            border-color: #667eea;
        }
        textarea {
            resize: vertical;
            min-height: 80px;
        }
        .info-box {
            background: #f0f7ff;
            border-left: 4px solid #667eea;
            padding: 15px;
            margin-bottom: 20px;
            border-radius: 4px;
        }
        .info-box p {
            color: #333;
            font-size: 14px;
            margin-bottom: 5px;
        }
        .info-box strong {
            color: #667eea;
        }
        button {
            width: 100%%;
            padding: 14px;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
            border: none;
            border-radius: 8px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: transform 0.2s;
        }
        button:hover {
            transform: translateY(-2px);
        }
        button:disabled {
            opacity: 0.6;
            cursor: not-allowed;
        }
        .message {
            margin-top: 20px;
            padding: 15px;
            border-radius: 8px;
            display: none;
        }
        .message.success {
            background: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
        }
        .message.error {
            background: #f8d7da;
            color: #721c24;
            border: 1px solid #f5c6cb;
        }
        .dns-info {
            margin-top: 20px;
            padding: 15px;
            background: #fff3cd;
            border-radius: 8px;
            display: none;
        }
        .dns-info h3 {
            color: #856404;
            margin-bottom: 10px;
            font-size: 16px;
        }
        .dns-info code {
            background: #ffeaa7;
            padding: 2px 6px;
            border-radius: 4px;
            font-family: 'Courier New', monospace;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🌐 DNS Access Registration</h1>
        <p class="subtitle">Register your IP address to access our DNS services</p>

        <div class="info-box">
            <p><strong>Your IP:</strong> <span id="userIP">%s</span></p>
            <p><strong>User:</strong> %s</p>
            <p><strong>Max IPs:</strong> %d</p>
            <p><strong>Current IPs:</strong> %d / %d</p>
            <p><strong>Expires:</strong> %s</p>
        </div>

        <form id="registerForm">
            <input type="hidden" name="token" value="%s">

            <button type="submit" id="submitBtn">Register This IP</button>
        </form>

        <div class="message" id="message"></div>

        <div class="dns-info" id="dnsInfo">
            <h3>✅ Registration Successful!</h3>
            <p>Configure your device to use our DNS:</p>
            <p><strong>DoH:</strong> <code id="dohURL"></code></p>
            <p><strong>DoT:</strong> <code id="dotServer"></code></p>
        </div>
    </div>

    <script>
        document.getElementById('registerForm').addEventListener('submit', async (e) => {
            e.preventDefault();

            const submitBtn = document.getElementById('submitBtn');
            const message = document.getElementById('message');
            const dnsInfo = document.getElementById('dnsInfo');

            submitBtn.disabled = true;
            submitBtn.textContent = 'Registering...';
            message.style.display = 'none';
            dnsInfo.style.display = 'none';

            const formData = {
                token: document.querySelector('[name="token"]').value
            };

            try {
                const response = await fetch('/register/submit', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(formData)
                });

                const result = await response.json();

                if (response.ok && result.success) {
                    message.className = 'message success';
                    message.textContent = 'Registration successful! You can now use our DNS services.';
                    message.style.display = 'block';

                    // Show DNS info
                    dnsInfo.style.display = 'block';
                    document.getElementById('dohURL').textContent = result.doh_url;
                    document.getElementById('dotServer').textContent = result.dot_server;

                    document.getElementById('registerForm').style.display = 'none';
                } else {
                    throw new Error(result.error || 'Registration failed');
                }
            } catch (error) {
                message.className = 'message error';
                message.textContent = error.message;
                message.style.display = 'block';
                submitBtn.disabled = false;
                submitBtn.textContent = 'Register';
            }
        });
    </script>
</body>
</html>
`, clientIP, user.Name, user.MaxIPs, len(user.IPs), user.MaxIPs, user.ExpiresAt.Format("2006-01-02 15:04"), token)

	ctx.SetContentType("text/html; charset=utf-8")
	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.WriteString(html)
}

func handleRegisterSubmit(ctx *fasthttp.RequestCtx) {
	var req struct {
		Token string `json:"token"` // User ID used as token
	}

	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.Error(`{"error":"Invalid JSON"}`, fasthttp.StatusBadRequest)
		return
	}

	if req.Token == "" {
		ctx.Error(`{"error":"Missing token"}`, fasthttp.StatusBadRequest)
		return
	}

	// Get client IP
	clientIP := getClientIP(ctx)

	// Add IP to user (with FIFO logic)
	err := addIPToUser(req.Token, clientIP)
	if err != nil {
		ctx.Error(fmt.Sprintf(`{"error":"%s"}`, err.Error()), fasthttp.StatusForbidden)
		return
	}

	// Get user info
	user := getUserByID(req.Token)
	if user == nil {
		ctx.Error(`{"error":"User not found"}`, fasthttp.StatusNotFound)
		return
	}

	cfg := getConfig()
	dohURL := fmt.Sprintf("https://%s/dns-query", cfg.Host)
	dotServer := fmt.Sprintf("%s:853", cfg.Host)

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(map[string]interface{}{
		"success":    true,
		"user":       user,
		"doh_url":    dohURL,
		"dot_server": dotServer,
		"expires_at": user.ExpiresAt,
	})
}

func runWebPanelServer(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	cfg := getConfig()
	if !cfg.WebPanelEnabled {
		logger.Info("web panel is disabled")
		return
	}

	if cfg.WebPanelUsername == "" || cfg.WebPanelPassword == "" {
		logger.Warn("web panel enabled but no credentials configured")
		return
	}

	server := &fasthttp.Server{
		Handler: func(c *fasthttp.RequestCtx) {
			path := string(c.Path())

			// CORS headers
			c.Response.Header.Set("Access-Control-Allow-Origin", "*")
			c.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			c.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type, X-Session-ID")

			if string(c.Method()) == "OPTIONS" {
				c.SetStatusCode(fasthttp.StatusOK)
				return
			}

			switch path {
			case "/panel", "/panel/":
				serveWebPanel(c)
			case "/panel/api/login":
				handlePanelLogin(c)
			case "/panel/api/logout":
				handlePanelLogout(c)
			case "/panel/api/validate":
				handlePanelValidate(c)
			case "/panel/api/metrics":
				handlePanelMetrics(c)
			case "/panel/api/health":
				handlePanelHealth(c)
			case "/panel/api/domains":
				handlePanelDomains(c)
			case "/panel/api/domains/add":
				handlePanelAddDomain(c)
			case "/panel/api/domains/remove":
				handlePanelRemoveDomain(c)
			case "/panel/api/reload":
				handlePanelReload(c)
			case "/panel/api/users":
				handlePanelUsers(c)
			case "/panel/api/users/create":
				handlePanelCreateUser(c)
			case "/panel/api/users/extend":
				handlePanelExtendUser(c)
			case "/panel/api/users/deactivate":
				handlePanelDeactivateUser(c)
			case "/panel/api/users/delete":
				handlePanelDeleteUser(c)
			case "/panel/api/settings/change-password":
				handlePanelChangePassword(c)
			case "/panel/api/settings/change-username":
				handlePanelChangeUsername(c)
			case "/register":
				serveRegisterPage(c)
			case "/register/submit":
				handleRegisterSubmit(c)
			default:
				c.Error("Not found", fasthttp.StatusNotFound)
			}
		},
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		logger.Info("web panel shutting down")
		_ = server.Shutdown()
	}()

	addr := fmt.Sprintf("0.0.0.0:%d", cfg.WebPanelPort)
	logger.Info("web panel started", "port", cfg.WebPanelPort, "url", fmt.Sprintf("http://0.0.0.0:%d/panel", cfg.WebPanelPort))

	if err := server.ListenAndServe(addr); err != nil {
		if ctx.Err() == nil {
			logger.Error("web panel server error", "error", err)
		}
	}
}

func runDOHServer(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	server := &fasthttp.Server{
		Handler: func(c *fasthttp.RequestCtx) {
			path := string(c.Path())
			switch path {
			case "/dns-query":
				handleDoHRequest(c)
			case "/health":
				handleHealthCheck(c)
			case "/metrics":
				handleMetrics(c)
			case "/metrics/prometheus":
				handlePrometheusMetrics(c)
			case "/admin/reload":
				handleConfigReload(c)
			default:
				c.Error("Unsupported path", fasthttp.StatusNotFound)
			}
		},
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		logger.Info("DoH server shutting down")
		_ = server.Shutdown()
	}()

	logger.Info("DoH server started", "address", "127.0.0.1:8080")
	if err := server.ListenAndServe("127.0.0.1:8080"); err != nil {
		if ctx.Err() == nil {
			logger.Error("DoH server error", "error", err)
			log.Fatalf("DoH server error: %v", err)
		}
	}
}

// ======================== main ========================

func main() {
	// Effective GC tuning at runtime (unlike setting env var)
	debug.SetGCPercent(50)

	// Load configuration
	cfg, err := LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	config.Store(cfg)

	// Initialize logger
	logger = initLogger(cfg.LogLevel)
	logger.Info("smartSNI starting", "version", "2.0")

	// Initialize upstream servers
	dohUpstream.Store(cfg.UpstreamDOH)

	// Load auth tokens
	for _, token := range cfg.AuthTokens {
		authTokens.Store(token, true)
	}

	// Optional override for DoH upstream via env
	if v := os.Getenv("DOH_UPSTREAM"); v != "" {
		dohURL = v
		logger.Info("DoH upstream override from env", "upstream", v)
	}

	// Shared rate limiter for DoH/DoT (50 req/s, burst 100)
	limiter = rate.NewLimiter(rate.Limit(50), 100)

	logger.Info("configuration loaded",
		"host", cfg.Host,
		"domains", len(cfg.Domains),
		"cache_ttl", cfg.CacheTTL,
		"upstream_servers", len(cfg.UpstreamDOH),
		"auth_enabled", cfg.EnableAuth,
		"metrics_enabled", cfg.MetricsEnabled,
	)

	// Start cache cleanup goroutine
	go cleanExpiredCache()

	// Start session cleanup goroutine
	go cleanExpiredSessions()

	// Setup signal handling
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// Start user expiration checker if user management is enabled
	if cfg.UserManagement {
		go startExpirationChecker(ctx)
		logger.Info("user expiration checker started")
	}
	defer stop()

	// Start all servers
	var wg sync.WaitGroup
	serverCount := 3

	// Check if DNS server should be started
	if cfg.DNSEnabled {
		serverCount++
	}

	// Check if web panel should be started
	if cfg.WebPanelEnabled && cfg.WebPanelUsername != "" && cfg.WebPanelPassword != "" {
		serverCount++
	}

	wg.Add(serverCount)

	go runDOHServer(ctx, &wg)
	go startDoTServer(ctx, &wg)
	go serveSniProxy(ctx, &wg)

	// Start DNS server if enabled
	if cfg.DNSEnabled {
		go startDNSServer(ctx, &wg)
	}

	// Start web panel if enabled
	if cfg.WebPanelEnabled && cfg.WebPanelUsername != "" && cfg.WebPanelPassword != "" {
		go runWebPanelServer(ctx, &wg)
	}

	sniPort := cfg.SNIPort
	if sniPort == 0 {
		sniPort = 443
	}

	logger.Info("all servers started",
		"dns_enabled", cfg.DNSEnabled,
		"sni_port", sniPort,
		"dot_port", 853,
		"doh_address", "127.0.0.1:8080",
		"web_panel_enabled", cfg.WebPanelEnabled,
		"web_panel_port", cfg.WebPanelPort,
	)

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info("shutdown signal received, stopping servers...")

	// Wait for all servers to stop
	wg.Wait()
	logger.Info("shutdown complete")
}
