package server

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/mamba-framework/mamba/framework/app"
    "github.com/mamba-framework/mamba/framework/config"
    "github.com/mamba-framework/mamba/framework/logger"
)

type Server struct {
    httpServer *http.Server
    app        *app.App
    config     *config.ServerConfig
    logger     logger.Logger
}

func New(cfg *config.ServerConfig, app *app.App, log logger.Logger) *Server {
    server := &Server{
        app:    app,
        config: cfg,
        logger: log,
    }

    server.httpServer = &http.Server{
        Addr:           fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
        Handler:        server.app.Router,
        ReadTimeout:    time.Duration(cfg.ReadTimeout) * time.Second,
        WriteTimeout:   time.Duration(cfg.WriteTimeout) * time.Second,
        MaxHeaderBytes: cfg.MaxHeaderBytes,
    }

    return server
}

func (s *Server) Start() error {
    s.logger.Info("Starting server...", 
        "addr", s.httpServer.Addr,
        "environment", s.app.Config.Environment)
    
    return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
    s.logger.Info("Shutting down server...")
    return s.httpServer.Shutdown(ctx)
}

func (s *Server) GetApp() *app.App {
    return s.app
}