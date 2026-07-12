package main

import (
    "flag"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/mamba-framework/mamba/framework/app"
    "github.com/mamba-framework/mamba/framework/config"
    "github.com/mamba-framework/mamba/framework/database"
    "github.com/mamba-framework/mamba/framework/logger"
    "github.com/mamba-framework/mamba/framework/server"
)

func main() {
    var configPath string
    flag.StringVar(&configPath, "config", "config/config.json", "Path to configuration file")
    flag.Parse()

    // Load configuration
    cfg, err := config.Load(configPath)
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }

    // Initialize logger
    log, err := logger.New(cfg.Logger)
    if err != nil {
        log.Fatalf("Failed to initialize logger: %v", err)
    }
    defer log.Sync()

    // Initialize database
    db, err := database.New(cfg.Database, log)
    if err != nil {
        log.Fatal("Failed to initialize database", "error", err)
    }
    defer db.Close()

    // Initialize application
    application := app.New(cfg, log, db)

    // Initialize server
    srv := server.New(cfg.Server, application, log)

    // Start server
    go func() {
        if err := srv.Start(); err != nil {
            log.Fatal("Server failed to start", "error", err)
        }
    }()

    log.Info("Server started successfully", 
        "host", cfg.Server.Host, 
        "port", cfg.Server.Port,
        "env", cfg.Environment)

    // Wait for shutdown signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Info("Shutting down server...")

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatal("Server shutdown failed", "error", err)
    }

    log.Info("Server stopped")
}
