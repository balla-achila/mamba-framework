package main

import (
    "fmt"
    "log"
    "net/http"

    "github.com/balla-achila/mamba-framework/framework/app"
    "github.com/balla-achila/mamba-framework/framework/config"
    "github.com/balla-achila/mamba-framework/framework/database"
    "github.com/balla-achila/mamba-framework/framework/logger"
    "github.com/balla-achila/mamba-framework/framework/router"
    "github.com/balla-achila/mamba-framework/framework/server"
    "github.com/balla-achila/mamba-framework/framework/session"
)

func main() {
    fmt.Println("========================================")
    fmt.Println("🚀 Mamba Framework - Complete Application")
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

    // Initialize database (NoOp for now, will use real DB later)
    db := database.NewNoOp()
    defer db.Close()

    // Initialize application
    application := app.New(cfg, log, db)

    // Add routes using the App context
    application.Router.Get("/", func(w http.ResponseWriter, r *http.Request) {
        html := `<!DOCTYPE html>
<html>
<head>
    <title>Mamba Framework</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; background: #f5f7fa; }
        .container { background: white; padding: 40px; border-radius: 10px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; margin-top: 0; }
        .logo { font-size: 3em; }
        .status { background: #28a745; color: white; padding: 10px; border-radius: 5px; text-align: center; }
        .endpoint { background: #f8f9fa; padding: 10px; margin: 5px 0; border-radius: 5px; font-family: monospace; }
        .badge { display: inline-block; padding: 2px 10px; background: #667eea; color: white; border-radius: 12px; font-size: 0.8em; }
        .footer { margin-top: 30px; text-align: center; color: #666; border-top: 1px solid #eee; padding-top: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="logo">🚀</div>
        <h1>Mamba Framework <span class="badge">v1.0</span></h1>
        <div class="status">✅ Framework is running!</div>
        
        <h3>📌 Available Endpoints:</h3>
        <div class="endpoint">GET / → This page</div>
        <div class="endpoint">GET /health → Health check</div>
        <div class="endpoint">GET /hello/:name → Say hello</div>
        <div class="endpoint">GET /session-test → Session test</div>
        <div class="endpoint">GET /api/users → Users API</div>
        
        <div class="footer">
            <p>Built with ❤️ using Mamba Framework</p>
            <p><small>Enterprise-Grade Go Framework</small></p>
        </div>
    </div>
</body>
</html>`
        w.Header().Set("Content-Type", "text/html")
        w.Write([]byte(html))
    })

    // Health check
    application.Router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"status":"ok","message":"Mamba Framework is healthy"}`))
    })

    // Hello endpoint
    application.Router.Get("/hello/:name", func(w http.ResponseWriter, r *http.Request) {
        name := router.GetRouteParam(r.Context(), "name")
        if name == "" {
            name = "World"
        }
        w.Write([]byte(fmt.Sprintf("Hello, %s! Welcome to Mamba Framework!", name)))
    })

    // Session test endpoint
    application.Router.Get("/session-test", func(w http.ResponseWriter, r *http.Request) {
        sess := session.FromContext(r.Context())
        if sess == nil {
            w.Write([]byte("Session not available"))
            return
        }
        
        count := 1
        if val := sess.Get("visit_count"); val != nil {
            if c, ok := val.(int); ok {
                count = c + 1
            }
        }
        sess.Set("visit_count", count)
        sess.Save()
        
        w.Write([]byte(fmt.Sprintf("You have visited this page %d times!", count)))
    })

    // API endpoint
    application.Router.Get("/api/users", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`[
            {"id":1,"name":"John Doe","email":"john@example.com","role":"admin"},
            {"id":2,"name":"Jane Smith","email":"jane@example.com","role":"user"},
            {"id":3,"name":"Bob Johnson","email":"bob@example.com","role":"user"}
        ]`))
    })

    // API POST example
    application.Router.Post("/api/users", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusCreated)
        w.Write([]byte(`{"message":"User created successfully","id":4}`))
    })

    // Initialize server
    srv := server.New(&cfg.Server, application, log)
    srv.SetHandler(application.Router)

    log.Info("🚀 Server starting", "addr", fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port))
    log.Info("📚 Visit http://localhost:%d", cfg.Server.Port)
    log.Info("💡 Press Ctrl+C to stop")

    if err := srv.Start(); err != nil && err != http.ErrServerClosed {
        log.Fatal("Server failed", "error", err)
    }
}
