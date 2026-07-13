package main

import (
    "fmt"
    "net/http"
    "os"

    "github.com/balla-achila/mamba-framework/framework/config"
    "github.com/balla-achila/mamba-framework/framework/logger"
    "github.com/balla-achila/mamba-framework/framework/router"
    "github.com/balla-achila/mamba-framework/framework/webmatrix"
)

func main() {
    fmt.Println("========================================")
    fmt.Println("🚀 Mamba Framework - WebMatrix Clone")
    fmt.Println("========================================")
    fmt.Println()

    cfg, err := config.Load("config/config.json")
    if err != nil {
        fmt.Printf("⚠️  Using default config: %v\n", err)
        cfg = config.DefaultConfig()
    }

    loggerCfg := &logger.Config{
        Level:      cfg.Logger.Level,
        OutputPath: cfg.Logger.OutputPath,
        MaxSize:    cfg.Logger.MaxSize,
        MaxBackups: cfg.Logger.MaxBackups,
        MaxAge:     cfg.Logger.MaxAge,
        Compress:   cfg.Logger.Compress,
    }

    log, err := logger.New(loggerCfg)
    if err != nil {
        fmt.Printf("Failed to initialize logger: %v\n", err)
        os.Exit(1)
    }
    defer log.Sync()

    // Initialize WebMatrix Engine
    wm := webmatrix.NewWebMatrixEngine(
        "templates/pages",
        "templates/layouts",
        "templates/partials",
    )

    r := router.New()

    // Home Page - Dashboard
    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        pageData := webmatrix.NewPageData("Dashboard")
        pageData.SetRequest(r)
        pageData.User = map[string]interface{}{
            "Username": "Admin",
            "Role":     "Administrator",
        }

        if err := wm.Render(w, "index", pageData); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
        }
    })

    // About Page
    r.Get("/about", func(w http.ResponseWriter, r *http.Request) {
        pageData := webmatrix.NewPageData("About")
        pageData.SetRequest(r)
        pageData.User = map[string]interface{}{
            "Username": "Admin",
            "Role":     "Administrator",
        }

        if err := wm.Render(w, "about", pageData); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
        }
    })

    // Employees List
    r.Get("/employees", func(w http.ResponseWriter, r *http.Request) {
        pageData := webmatrix.NewPageData("Employees")
        pageData.SetRequest(r)
        pageData.User = map[string]interface{}{
            "Username": "Admin",
            "Role":     "Administrator",
        }

        if err := wm.Render(w, "employees", pageData); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
        }
    })

    // Employee Create - GET
    r.Get("/employees/create", func(w http.ResponseWriter, r *http.Request) {
        pageData := webmatrix.NewPageData("Create Employee")
        pageData.SetRequest(r)
        pageData.User = map[string]interface{}{
            "Username": "Admin",
            "Role":     "Administrator",
        }

        if err := wm.Render(w, "employee-create", pageData); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
        }
    })

    // Employee Create - POST
    r.Post("/employees/create", func(w http.ResponseWriter, r *http.Request) {
        pageData := webmatrix.NewPageData("Create Employee")
        pageData.SetRequest(r)
        pageData.User = map[string]interface{}{
            "Username": "Admin",
            "Role":     "Administrator",
        }

        if err := wm.Render(w, "employee-create", pageData); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
        }
    })

    // Health Check
    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"status":"ok","message":"Mamba Framework is healthy"}`))
    })

    addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
    log.Info("🚀 Server starting", "addr", addr)
    log.Info("📚 Demo Pages:")
    log.Info("   - Home: http://localhost:%d/", cfg.Server.Port)
    log.Info("   - About: http://localhost:%d/about", cfg.Server.Port)
    log.Info("   - Employees: http://localhost:%d/employees", cfg.Server.Port)
    log.Info("   - Create Employee: http://localhost:%d/employees/create", cfg.Server.Port)
    log.Info("💡 Press Ctrl+C to stop")

    if err := http.ListenAndServe(addr, r); err != nil {
        log.Fatal("Server failed", "error", err)
    }
}
