package server

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/balla-achila/mamba-framework/framework/config"
    "github.com/balla-achila/mamba-framework/framework/logger"
)

type Server struct {
    httpServer *http.Server
    app        interface{}
    config     *config.ServerConfig
    logger     logger.Logger
}

func New(cfg *config.ServerConfig, app interface{}, log logger.Logger) *Server {
    server := &Server{
        app:    app,
        config: cfg,
        logger: log,
    }

    server.httpServer = &http.Server{
        Addr:           fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
        ReadTimeout:    time.Duration(cfg.ReadTimeout) * time.Second,
        WriteTimeout:   time.Duration(cfg.WriteTimeout) * time.Second,
        MaxHeaderBytes: cfg.MaxHeaderBytes,
    }

    return server
}

func (s *Server) SetHandler(handler http.Handler) {
    s.httpServer.Handler = handler
}

func (s *Server) Start() error {
    s.logger.Info("Starting server...",
        "addr", s.httpServer.Addr,
    )

    return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
    s.logger.Info("Shutting down server...")
    return s.httpServer.Shutdown(ctx)
}
