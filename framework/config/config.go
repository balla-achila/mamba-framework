package config

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
)

type Config struct {
    Environment string          `json:"environment"`
    Server      ServerConfig    `json:"server"`
    Database    DatabaseConfig  `json:"database"`
    Logger      LoggerConfig    `json:"logger"`
    Session     SessionConfig   `json:"session"`
    Security    SecurityConfig  `json:"security"`
    Upload      UploadConfig    `json:"upload"`
    Tenant      TenantConfig    `json:"tenant"`
}

type ServerConfig struct {
    Host            string   `json:"host"`
    Port            int      `json:"port"`
    ReadTimeout     int      `json:"read_timeout"`
    WriteTimeout    int      `json:"write_timeout"`
    MaxHeaderBytes  int      `json:"max_header_bytes"`
    TemplatesPath   string   `json:"templates_path"`
    StaticPath      string   `json:"static_path"`
    UploadsPath     string   `json:"uploads_path"`
}

type DatabaseConfig struct {
    Host            string `json:"host"`
    Port            int    `json:"port"`
    User            string `json:"user"`
    Password        string `json:"password"`
    Database        string `json:"database"`
    SSLMode         string `json:"ssl_mode"`
    MaxConnections  int    `json:"max_connections"`
    MinConnections  int    `json:"min_connections"`
    MaxIdleTime     int    `json:"max_idle_time"`
    MaxLifeTime     int    `json:"max_life_time"`
    QueryTimeout    int    `json:"query_timeout"`
}

type LoggerConfig struct {
    Level      string `json:"level"`
    OutputPath string `json:"output_path"`
    MaxSize    int    `json:"max_size"`
    MaxBackups int    `json:"max_backups"`
    MaxAge     int    `json:"max_age"`
    Compress   bool   `json:"compress"`
}

type SessionConfig struct {
    SecretKey string `json:"secret_key"`
    Name      string `json:"name"`
    MaxAge    int    `json:"max_age"`
    Secure    bool   `json:"secure"`
    HttpOnly  bool   `json:"http_only"`
    SameSite  string `json:"same_site"`
}

type SecurityConfig struct {
    CSRFSecret      string   `json:"csrf_secret"`
    CSRFMaxAge      int      `json:"csrf_max_age"`
    RateLimit       int      `json:"rate_limit"`
    RateLimitWindow int      `json:"rate_limit_window"`
    AllowedHosts    []string `json:"allowed_hosts"`
    ForceHTTPS      bool     `json:"force_https"`
    HSTSMaxAge      int      `json:"hsts_max_age"`
}

type UploadConfig struct {
    MaxSize      int64    `json:"max_size"`
    AllowedTypes []string `json:"allowed_types"`
    TempDir      string   `json:"temp_dir"`
    Permissions  string   `json:"permissions"`
    ChunkSize    int64    `json:"chunk_size"`
}

type TenantConfig struct {
    Enabled         bool   `json:"enabled"`
    DefaultTenantID string `json:"default_tenant_id"`
    TenantHeader    string `json:"tenant_header"`
    TenantParam     string `json:"tenant_param"`
}

func Load(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return DefaultConfig(), fmt.Errorf("failed to read config file: %w", err)
    }

    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("failed to parse config file: %w", err)
    }

    // Set defaults if not specified
    if cfg.Server.TemplatesPath == "" {
        cfg.Server.TemplatesPath = "templates"
    }
    if cfg.Server.StaticPath == "" {
        cfg.Server.StaticPath = "static"
    }
    if cfg.Server.UploadsPath == "" {
        cfg.Server.UploadsPath = "uploads"
    }
    if cfg.Server.MaxHeaderBytes == 0 {
        cfg.Server.MaxHeaderBytes = 1 << 20
    }
    if cfg.Database.MaxConnections == 0 {
        cfg.Database.MaxConnections = 25
    }
    if cfg.Database.MinConnections == 0 {
        cfg.Database.MinConnections = 5
    }
    if cfg.Database.QueryTimeout == 0 {
        cfg.Database.QueryTimeout = 30
    }
    if cfg.Session.MaxAge == 0 {
        cfg.Session.MaxAge = 86400
    }
    if cfg.Security.RateLimit == 0 {
        cfg.Security.RateLimit = 100
    }
    if cfg.Security.RateLimitWindow == 0 {
        cfg.Security.RateLimitWindow = 60
    }
    if cfg.Upload.MaxSize == 0 {
        cfg.Upload.MaxSize = 10 << 20
    }

    return &cfg, nil
}

func DefaultConfig() *Config {
    return &Config{
        Environment: "development",
        Server: ServerConfig{
            Host:           "0.0.0.0",
            Port:           8080,
            ReadTimeout:    15,
            WriteTimeout:   15,
            MaxHeaderBytes: 1 << 20,
            TemplatesPath:  "templates",
            StaticPath:     "static",
            UploadsPath:    "uploads",
        },
        Database: DatabaseConfig{
            Host:           "localhost",
            Port:           5432,
            User:           "mamba",
            Password:       "mamba_password",
            Database:       "mamba",
            SSLMode:        "disable",
            MaxConnections: 25,
            MinConnections: 5,
            MaxIdleTime:    300,
            MaxLifeTime:    3600,
            QueryTimeout:   30,
        },
        Logger: LoggerConfig{
            Level:      "debug",
            OutputPath: "stdout",
            MaxSize:    100,
            MaxBackups: 5,
            MaxAge:     30,
            Compress:   true,
        },
        Session: SessionConfig{
            SecretKey: "change-this-in-production-32-characters",
            Name:      "mamba_session",
            MaxAge:    86400,
            Secure:    false,
            HttpOnly:  true,
            SameSite:  "lax",
        },
        Security: SecurityConfig{
            CSRFSecret:      "",
            CSRFMaxAge:      86400,
            RateLimit:       100,
            RateLimitWindow: 60,
            AllowedHosts:    []string{"localhost", "127.0.0.1"},
            ForceHTTPS:      false,
            HSTSMaxAge:      0,
        },
        Upload: UploadConfig{
            MaxSize: 10 << 20,
            AllowedTypes: []string{
                "image/jpeg", "image/png", "image/gif",
            },
            TempDir:      "/tmp/mamba_uploads",
            Permissions:  "0644",
            ChunkSize:    1048576,
        },
        Tenant: TenantConfig{
            Enabled:         true,
            DefaultTenantID: "default",
            TenantHeader:    "X-Tenant-ID",
            TenantParam:     "tenant",
        },
    }
}

func (c *Config) IsProduction() bool {
    return c.Environment == "production"
}

func (c *Config) IsDevelopment() bool {
    return c.Environment == "development"
}

func (c *Config) GetDSN() string {
    return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
        c.Database.Host, c.Database.Port, c.Database.User,
        c.Database.Password, c.Database.Database, c.Database.SSLMode)
}