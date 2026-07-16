package security

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/balla-achila/mamba-framework/framework/config"
	"github.com/balla-achila/mamba-framework/framework/logger"
	"github.com/gorilla/csrf"
)

type Security struct {
	config         *config.SecurityConfig
	logger         logger.Logger
	csrfMiddleware func(http.Handler) http.Handler
	rateLimiter    *RateLimiter
	mu             sync.RWMutex
}

type RateLimiter struct {
	requests map[string][]time.Time
	limit    int
	window   int64
	mu       sync.RWMutex
}

func New(cfg *config.Config, log logger.Logger) *Security {
	sec := &Security{
		config: &cfg.Security,
		logger: log,
	}

	sec.setupCSRF()

	sec.rateLimiter = &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    cfg.Security.RateLimit,
		window:   int64(cfg.Security.RateLimitWindow),
	}

	return sec
}

func (s *Security) setupCSRF() {
	if s.config.CSRFSecret == "" {
		s.config.CSRFSecret = generateRandomString(32)
	}

	csrfOptions := []csrf.Option{
		csrf.MaxAge(s.config.CSRFMaxAge),
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "CSRF token validation failed", http.StatusForbidden)
		})),
	}

	if s.config.ForceHTTPS {
		csrfOptions = append(csrfOptions, csrf.Secure(true))
	}

	s.csrfMiddleware = csrf.Protect(
		[]byte(s.config.CSRFSecret),
		csrfOptions...,
	)
}

func (s *Security) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Check allowed hosts - skip if in development or no hosts configured
		if len(s.config.AllowedHosts) > 0 {
			allowed := false
			for _, host := range s.config.AllowedHosts {
				// Check if host matches (including port)
				if r.Host == host || strings.HasPrefix(r.Host, host+":") {
					allowed = true
					break
				}
			}
			if !allowed {
				s.logger.Warn("Invalid host rejected", "host", r.Host, "allowed_hosts", s.config.AllowedHosts)
				http.Error(w, "Invalid host", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Security) CSRFMiddleware(next http.Handler) http.Handler {
	return s.csrfMiddleware(next)
}

func (s *Security) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}

		clientIP := s.getClientIP(r)

		if !s.rateLimiter.Allow(clientIP) {
			s.logger.Warn("Rate limit exceeded", "ip", clientIP, "path", r.URL.Path)
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Security) GetCSRFToken(r *http.Request) string {
	return csrf.Token(r)
}

// getClientIP returns the request's real client IP. X-Forwarded-For and
// X-Real-IP are only trusted when the immediate peer (r.RemoteAddr) is in
// the configured TrustedProxies list -- any client can set these headers
// themselves, so trusting them unconditionally lets an attacker forge a
// different IP on every request and bypass the rate limiter entirely.
func (s *Security) getClientIP(r *http.Request) string {
	remoteIP := r.RemoteAddr
	if strings.Contains(remoteIP, ":") {
		remoteIP = strings.Split(remoteIP, ":")[0]
	}

	trusted := false
	for _, p := range s.config.TrustedProxies {
		if p == remoteIP {
			trusted = true
			break
		}
	}
	if !trusted {
		return remoteIP
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	return remoteIP
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-time.Duration(rl.window) * time.Second)

	if requests, ok := rl.requests[key]; ok {
		valid := make([]time.Time, 0, len(requests))
		for _, t := range requests {
			if t.After(windowStart) {
				valid = append(valid, t)
			}
		}
		rl.requests[key] = valid
	}

	if len(rl.requests[key]) >= rl.limit {
		return false
	}

	rl.requests[key] = append(rl.requests[key], now)
	return true
}

func generateRandomString(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failing means the OS entropy source is broken; there's
		// no safe fallback for a secret in that case.
		panic("security: failed to generate random string: " + err.Error())
	}
	return base64.URLEncoding.EncodeToString(b)
}

func (s *Security) SanitizeInput(input string) string {
	input = strings.ReplaceAll(input, "&", "&amp;")
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, "\"", "&quot;")
	input = strings.ReplaceAll(input, "'", "&#x27;")
	return input
}
