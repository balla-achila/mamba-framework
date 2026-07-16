package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver – replace with "github.com/jackc/pgx/v5/stdlib" if using pgx

	"github.com/balla-achila/mamba-framework/framework/config"
	"github.com/balla-achila/mamba-framework/framework/logger"
	"github.com/balla-achila/mamba-framework/framework/router"
	"github.com/balla-achila/mamba-framework/framework/security"
	"github.com/balla-achila/mamba-framework/framework/webmatrix"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("🚀 Mamba Framework - WebMatrix Clone")
	fmt.Println("========================================")
	fmt.Println()

	// ------------------------------
	// 1. Load Configuration
	// ------------------------------
	cfg, err := config.Load("config/config.json")
	if err != nil {
		fmt.Printf("⚠️  Using default config: %v\n", err)
		cfg = config.DefaultConfig()
	}

	// ------------------------------
	// 2. Setup Logger
	// ------------------------------
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

	// ------------------------------
	// 3. Initialize WebMatrix Engine
	// ------------------------------
	wm := webmatrix.NewWebMatrixEngine(
		"templates/pages",    // pages directory
		"templates/layouts",  // layouts directory
		"templates/partials", // partials directory
	)

	// ------------------------------
	// 4. Setup Database
	// ------------------------------
	// Get connection string from web.config (or use default)
	connStr := wm.ConnectionString("DefaultConnection")
	if connStr == "" {
		// fallback to config or default
		connStr = "postgres://user:pass@localhost/mambadb?sslmode=disable"
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Warn("Failed to open database", "error", err)
		log.Warn("Database features will use fallback sample data")
	} else {
		// Ping to verify connection
		if err := db.Ping(); err != nil {
			log.Warn("Database ping failed", "error", err)
			log.Warn("Database features will use fallback sample data")
		} else {
			// Register the connection with the engine
			wm.SetDB("DefaultConnection", db)
			log.Info("Database connected successfully")
		}
	}
	// Note: db will be closed on shutdown (defer)
	defer func() {
		if db != nil {
			db.Close()
		}
	}()

	// ------------------------------
	// 5. Setup Router + Security
	// ------------------------------
	r := router.New()

	// Security headers, host allowlist, and rate limiting. CSRF is handled
	// separately below via webmatrix's own token (VerifyAntiForgeryToken),
	// rather than security.CSRFMiddleware, to avoid running two different
	// CSRF schemes at once.
	sec := security.New(cfg, log)
	r.Use(sec.Middleware)
	r.Use(sec.RateLimitMiddleware)

	// --- Home Page (Dashboard) ---
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		pageData := webmatrix.NewPageData("Dashboard")
		pageData.SetRequest(r)
		pageData.User = map[string]interface{}{
			"Username": "Admin",
			"Role":     "Administrator",
		}

		if err := wm.Render(w, "index", pageData); err != nil {
			log.Error("Failed to render index", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// --- About Page ---
	r.Get("/about", func(w http.ResponseWriter, r *http.Request) {
		pageData := webmatrix.NewPageData("About Mamba Framework")
		pageData.SetRequest(r)
		pageData.User = map[string]interface{}{
			"Username": "Admin",
			"Role":     "Administrator",
		}

		if err := wm.Render(w, "about", pageData); err != nil {
			log.Error("Failed to render about", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// --- Employees List ---
	r.Get("/employees", func(w http.ResponseWriter, r *http.Request) {
		pageData := webmatrix.NewPageData("Employees")
		pageData.SetRequest(r)
		pageData.User = map[string]interface{}{
			"Username": "Admin",
			"Role":     "Administrator",
		}

		if err := wm.Render(w, "employees", pageData); err != nil {
			log.Error("Failed to render employees", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// --- Employee Create (GET) ---
	r.Get("/employees/create", func(w http.ResponseWriter, r *http.Request) {
		pageData := webmatrix.NewPageData("Create Employee")
		pageData.SetRequest(r)
		pageData.User = map[string]interface{}{
			"Username": "Admin",
			"Role":     "Administrator",
		}

		if err := wm.Render(w, "employee-create", pageData); err != nil {
			log.Error("Failed to render employee-create", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// --- Employee Create (POST) ---
	r.Post("/employees/create", func(w http.ResponseWriter, r *http.Request) {
		if !webmatrix.VerifyAntiForgeryToken(r) {
			log.Warn("CSRF token validation failed", "path", r.URL.Path)
			http.Error(w, "CSRF token validation failed", http.StatusForbidden)
			return
		}

		pageData := webmatrix.NewPageData("Create Employee")
		pageData.SetRequest(r)
		pageData.User = map[string]interface{}{
			"Username": "Admin",
			"Role":     "Administrator",
		}

		if err := wm.Render(w, "employee-create", pageData); err != nil {
			log.Error("Failed to render employee-create (POST)", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// --- Health Check (for monitoring) ---
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
	})

	// --- Static Files (optional) ---
	r.Get("/static/*", func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/static/", http.FileServer(http.Dir("static"))).ServeHTTP(w, r)
	})

	// ------------------------------
	// 6. Start Server
	// ------------------------------
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// Print startup info
	log.Info("🚀 Server starting", "addr", addr)
	log.Info("📚 Demo Pages:")
	log.Info("   - Home: http://localhost:%d/", cfg.Server.Port)
	log.Info("   - About: http://localhost:%d/about", cfg.Server.Port)
	log.Info("   - Employees: http://localhost:%d/employees", cfg.Server.Port)
	log.Info("   - Create Employee: http://localhost:%d/employees/create", cfg.Server.Port)
	log.Info("💡 Press Ctrl+C to stop")

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed", "error", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server shutdown failed", "error", err)
	}

	log.Info("Server stopped gracefully")
}
