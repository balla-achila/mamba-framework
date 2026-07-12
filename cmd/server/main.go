package main

import (
    "fmt"
    "log"
    "net/http"

    "github.com/balla-achila/mamba-framework/framework/config"
    "github.com/balla-achila/mamba-framework/framework/database"
    "github.com/balla-achila/mamba-framework/framework/logger"
    "github.com/balla-achila/mamba-framework/framework/router"
    "github.com/balla-achila/mamba-framework/framework/security"
    "github.com/balla-achila/mamba-framework/framework/server"
    "github.com/balla-achila/mamba-framework/framework/session"
)

func main() {
    fmt.Println("========================================")
    fmt.Println("🚀 Mamba Framework - Full Build")
    fmt.Println("========================================")
    fmt.Println()

    // Load config
    cfg, err := config.Load("config/config.json")
    if err != nil {
        log.Printf("⚠️  Using default config: %v", err)
        cfg = config.DefaultConfig()
    }

    // Initialize logger
    log, err := logger.New(&cfg.Logger)
    if err != nil {
        log.Fatalf("Failed to initialize logger: %v", err)
    }
    defer log.Sync()

    // Initialize database (NoOp for now)
    db := database.NewNoOp()
    defer db.Close()

    // Initialize session manager
    sessionCfg := &session.Config{
        SecretKey: cfg.Session.SecretKey,
        Name:      cfg.Session.Name,
        MaxAge:    cfg.Session.MaxAge,
        Secure:    cfg.Session.Secure,
        HttpOnly:  cfg.Session.HttpOnly,
        SameSite:  cfg.Session.SameSite,
    }
    sessionMgr := session.New(sessionCfg)

    // Initialize security
    securityMgr := security.New(cfg, log)

    // Initialize router
    r := router.New()

    // Add middleware
    r.Use(sessionMgr.Middleware)
    r.Use(securityMgr.Middleware)
    r.Use(securityMgr.CSRFMiddleware)
    r.Use(securityMgr.RateLimitMiddleware)

    // Add test routes
    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Mamba Framework</title></head>
<body style="font-family: sans-serif; max-width: 800px; margin: 50px auto; padding: 20px;">
    <h1>🚀 Mamba Framework</h1>
    <p>Framework is running successfully!</p>
    <h3>Available Endpoints:</h3>
    <ul>
        <li><a href="/">/</a> - This page</li>
        <li><a href="/health">/health</a> - Health check</li>
        <li><a href="/hello/World">/hello/:name</a> - Say hello</li>
        <li><a href="/session-test">/session-test</a> - Session test</li>
    </ul>
</body>
</html>`))
    })

    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"status":"ok","message":"Mamba Framework is healthy"}`))
    })

    r.Get("/hello/:name", func(w http.ResponseWriter, r *http.Request) {
        name := router.GetRouteParam(r.Context(), "name")
        if name == "" {
            name = "World"
        }
        w.Write([]byte(fmt.Sprintf("Hello, %s! Welcome to Mamba Framework!", name)))
    })

    r.Get("/session-test", func(w http.ResponseWriter, r *http.Request) {
        sess := session.FromContext(r.Context())
        if sess != nil {
            count := 1
            if val := sess.Get("visit_count"); val != nil {
                if c, ok := val.(int); ok {
                    count = c + 1
                }
            }
            sess.Set("visit_count", count)
            sess.Save()
            w.Write([]byte(fmt.Sprintf("You have visited this page %d times!", count)))
        } else {
            w.Write([]byte("Session not available"))
        }
    })

    // Initialize server
    srv := server.New(&cfg.Server, nil, log)
    srv.SetHandler(r)

    log.Info("🚀 Server starting on", "addr", fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port))
    log.Info("📚 Visit http://localhost:%d", cfg.Server.Port)
    log.Info("💡 Press Ctrl+C to stop")

    if err := srv.Start(); err != nil && err != http.ErrServerClosed {
        log.Fatal("Server failed", "error", err)
    }
}
