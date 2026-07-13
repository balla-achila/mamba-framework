package logger

import (
    "fmt"
    "io"
    "os"
    "time"

    "github.com/rs/zerolog"
)

// Logger interface defines logging methods
type Logger interface {
    Debug(msg string, args ...interface{})
    Info(msg string, args ...interface{})
    Warn(msg string, args ...interface{})
    Error(msg string, args ...interface{})
    Fatal(msg string, args ...interface{})
    Fatalf(format string, args ...interface{})
    With(args ...interface{}) Logger
    Sync() error
}

// Config holds logger configuration
type Config struct {
    Level      string `json:"level"`
    OutputPath string `json:"output_path"`
    MaxSize    int    `json:"max_size"`
    MaxBackups int    `json:"max_backups"`
    MaxAge     int    `json:"max_age"`
    Compress   bool   `json:"compress"`
}

// MambaLogger implements the Logger interface
type MambaLogger struct {
    logger zerolog.Logger
    config *Config
}

// New creates a new logger instance
func New(cfg *Config) (Logger, error) {
    level, err := zerolog.ParseLevel(cfg.Level)
    if err != nil {
        level = zerolog.InfoLevel
    }
    zerolog.SetGlobalLevel(level)

    var output io.Writer
    if cfg.OutputPath == "" || cfg.OutputPath == "stdout" {
        output = zerolog.ConsoleWriter{
            Out:        os.Stdout,
            TimeFormat: time.RFC3339,
        }
    } else {
        file, err := os.OpenFile(cfg.OutputPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
        if err != nil {
            return nil, fmt.Errorf("failed to open log file: %w", err)
        }
        output = file
    }

    logger := zerolog.New(output).
        With().
        Timestamp().
        Caller().
        Logger()

    return &MambaLogger{
        logger: logger,
        config: cfg,
    }, nil
}

// Debug logs a debug message
func (l *MambaLogger) Debug(msg string, args ...interface{}) {
    if len(args) > 0 {
        l.logger.Debug().Fields(argsToFields(args...)).Msg(msg)
    } else {
        l.logger.Debug().Msg(msg)
    }
}

// Info logs an info message
func (l *MambaLogger) Info(msg string, args ...interface{}) {
    if len(args) > 0 {
        l.logger.Info().Fields(argsToFields(args...)).Msg(msg)
    } else {
        l.logger.Info().Msg(msg)
    }
}

// Warn logs a warning message
func (l *MambaLogger) Warn(msg string, args ...interface{}) {
    if len(args) > 0 {
        l.logger.Warn().Fields(argsToFields(args...)).Msg(msg)
    } else {
        l.logger.Warn().Msg(msg)
    }
}

// Error logs an error message
func (l *MambaLogger) Error(msg string, args ...interface{}) {
    if len(args) > 0 {
        l.logger.Error().Fields(argsToFields(args...)).Msg(msg)
    } else {
        l.logger.Error().Msg(msg)
    }
}

// Fatal logs a fatal message and exits
func (l *MambaLogger) Fatal(msg string, args ...interface{}) {
    if len(args) > 0 {
        l.logger.Fatal().Fields(argsToFields(args...)).Msg(msg)
    } else {
        l.logger.Fatal().Msg(msg)
    }
}

// Fatalf logs a formatted fatal message and exits
func (l *MambaLogger) Fatalf(format string, args ...interface{}) {
    l.logger.Fatal().Msgf(format, args...)
}

// With adds fields to the logger
func (l *MambaLogger) With(args ...interface{}) Logger {
    if len(args) > 0 {
        return &MambaLogger{
            logger: l.logger.With().Fields(argsToFields(args...)).Logger(),
            config: l.config,
        }
    }
    return l
}

// Sync flushes any buffered log entries
func (l *MambaLogger) Sync() error {
    return nil
}

// argsToFields converts variadic arguments to a field map
func argsToFields(args ...interface{}) map[string]interface{} {
    fields := make(map[string]interface{})
    for i := 0; i < len(args)-1; i += 2 {
        if key, ok := args[i].(string); ok {
            fields[key] = args[i+1]
        }
    }
    return fields
}
