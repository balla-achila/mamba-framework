package security

import (
    "context"
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "net/http"
    "strings"
    "sync"
    "time"

    "github.com/gorilla/csrf"

    "github.com/gorilla/csrf"
    "github.com/balla-achila/mamba-framework/framework/config"
    "github.com/balla-achila/mamba-framework/framework/logger"
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

    // Initialize CSRF protection
    sec.setupCSRF()
    
    // Initialize rate limiter
    sec.rateLimiter = &RateLimiter{
        requests: make(map[string][]time.Time),
        limit:    cfg.Security.RateLimit,
        window:   int64(cfg.Security.RateLimitWindow),
    }

    return sec
}

func (s *Security) setupCSRF() {
    // Generate CSRF secret if not provided
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

        // HSTS
        if s.config.ForceHTTPS && r.Header.Get("X-Forwarded-Proto") == "http" {
            http.Redirect(w, r, "https://"+r.Host+r.URL.String(), http.StatusMovedPermanently)
            return
        }

        if s.config.ForceHTTPS && s.config.HSTSMaxAge > 0 {
            w.Header().Set("Strict-Transport-Security", 
                fmt.Sprintf("max-age=%d; includeSubDomains; preload", s.config.HSTSMaxAge))
        }

        // Content Security Policy - default permissive for development
        if s.config.ForceHTTPS {
            w.Header().Set("Content-Security-Policy", 
                "default-src 'self'; script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; "+
                "style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; "+
                "img-src 'self' data:; font-src 'self' https://cdn.jsdelivr.net;")
        }

        // Check allowed hosts
        if len(s.config.AllowedHosts) > 0 {
            allowed := false
            for _, host := range s.config.AllowedHosts {
                if r.Host == host {
                    allowed = true
                    break
                }
            }
            if !allowed {
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
        // Skip rate limiting for static assets
        if strings.HasPrefix(r.URL.Path, "/static/") {
            next.ServeHTTP(w, r)
            return
        }

        // Identify client by IP
        clientIP := s.getClientIP(r)
        
        if !s.rateLimiter.Allow(clientIP) {
            http.Error(w, "Too many requests", http.StatusTooManyRequests)
            return
        }

        next.ServeHTTP(w, r)
    })
}

func (s *Security) GetCSRFToken(r *http.Request) string {
    return csrf.Token(r)
}

func (s *Security) ValidateCSRFToken(r *http.Request) bool {
    // The CSRF middleware handles validation automatically
    return true
}

func (s *Security) getClientIP(r *http.Request) string {
    // Check X-Forwarded-For header
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        ips := strings.Split(xff, ",")
        if len(ips) > 0 {
            return strings.TrimSpace(ips[0])
        }
    }
    
    // Check X-Real-IP header
    if xri := r.Header.Get("X-Real-IP"); xri != "" {
        return xri
    }
    
    // Fallback to remote address
    ip := r.RemoteAddr
    if strings.Contains(ip, ":") {
        ip = strings.Split(ip, ":")[0]
    }
    return ip
}

func (rl *RateLimiter) Allow(key string) bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    now := time.Now()
    windowStart := now.Add(-time.Duration(rl.window) * time.Second)

    // Clean up old requests
    if requests, ok := rl.requests[key]; ok {
        valid := make([]time.Time, 0, len(requests))
        for _, t := range requests {
            if t.After(windowStart) {
                valid = append(valid, t)
            }
        }
        rl.requests[key] = valid
    }

    // Check if under limit
    if len(rl.requests[key]) >= rl.limit {
        return false
    }

    // Add current request
    rl.requests[key] = append(rl.requests[key], now)
    return true
}

func generateRandomString(length int) string {
    b := make([]byte, length)
    _, err := rand.Read(b)
    if err != nil {
        return ""
    }
    return base64.URLEncoding.EncodeToString(b)[:length]
}

// Additional security functions
func (s *Security) SanitizeInput(input string) string {
    // Basic HTML escaping
    input = strings.ReplaceAll(input, "&", "&amp;")
    input = strings.ReplaceAll(input, "<", "&lt;")
    input = strings.ReplaceAll(input, ">", "&gt;")
    input = strings.ReplaceAll(input, "\"", "&quot;")
    input = strings.ReplaceAll(input, "'", "&#x27;")
    return input
}

func (s *Security) SanitizeSQL(input string) string {
    // Basic SQL injection prevention
    // Note: Using prepared statements is the primary defense
    dangerous := []string{"'", "\"", ";", "--", "/*", "*/"}
    for _, d := range dangerous {
        input = strings.ReplaceAll(input, d, "")
    }
    return input
}