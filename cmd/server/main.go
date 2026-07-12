package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/balla-achila/mamba-framework/framework/config"
	"github.com/balla-achila/mamba-framework/framework/logger"
	"github.com/balla-achila/mamba-framework/framework/router"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("🚀 Mamba Framework - Basic Test")
	fmt.Println("========================================")
	fmt.Println()

	// Load config
	cfg, err := config.Load("config/config.json")
	if err != nil {
		log.Printf("⚠️  Using default config: %v", err)
		cfg = config.DefaultConfig()
	}

	// Initialize custom logger by mapping fields explicitly to prevent type mismatches
	appLogger, err := logger.New(&logger.Config{
		Level:      cfg.Logger.Level,
		OutputPath: cfg.Logger.OutputPath,
		MaxSize:    cfg.Logger.MaxSize,
		MaxBackups: cfg.Logger.MaxBackups,
		MaxAge:     cfg.Logger.MaxAge,
		Compress:   cfg.Logger.Compress,
	})
	if err != nil {
		// Using standard log package here since appLogger failed to initialize
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer appLogger.Sync()

	// Initialize router
	r := router.New()

	// Add test routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Mamba Framework is running!"))
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

	// Start server (ensuring host and port are formatted with a colon separating them)
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	appLogger.Info("🚀 Server starting on " + addr)
	appLogger.Info("📚 Visit http://" + addr)
	appLogger.Info("💡 Press Ctrl+C to stop")
	fmt.Println()

	if err := http.ListenAndServe(addr, r); err != nil {
		appLogger.Fatal("Server failed", "error", err)
	}
}