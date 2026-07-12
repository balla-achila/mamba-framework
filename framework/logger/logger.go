package logger

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Fatal(msg string, args ...interface{})
	With(args ...interface{}) Logger
	Sync() error
}

type Config struct {
	Level      string `json:"level"`
	OutputPath string `json:"output_path"`
	MaxSize    int    `json:"max_size"`
	MaxBackups int    `json:"max_backups"`
	MaxAge     int    `json:"max_age"`
	Compress   bool   `json:"compress"`
}

type MambaLogger struct {
	logger zerolog.Logger
	config *Config
	closer io.Closer // Kept to safely flush/sync files if needed
}

func New(cfg *Config) (Logger, error) {
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// CRITICAL: Tells zerolog to skip wrapper frames 
	// so it prints the real caller's file and line number.
	zerolog.CallerSkipFrameCount = 2

	var output io.Writer
	var closer io.Closer

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
		closer = file
	}

	logger := zerolog.New(output).
		With().
		Timestamp().
		Caller().
		Logger()

	return &MambaLogger{
		logger: logger,
		config: cfg,
		closer: closer,
	}, nil
}

func (l *MambaLogger) Debug(msg string, args ...interface{}) {
	if len(args) > 0 {
		l.logger.Debug().Fields(argsToFields(args...)).Msg(msg)
	} else {
		l.logger.Debug().Msg(msg)
	}
}

func (l *MambaLogger) Info(msg string, args ...interface{}) {
	if len(args) > 0 {
		l.logger.Info().Fields(argsToFields(args...)).Msg(msg)
	} else {
		l.logger.Info().Msg(msg)
	}
}

func (l *MambaLogger) Warn(msg string, args ...interface{}) {
	if len(args) > 0 {
		l.logger.Warn().Fields(argsToFields(args...)).Msg(msg)
	} else {
		l.logger.Warn().Msg(msg)
	}
}

func (l *MambaLogger) Error(msg string, args ...interface{}) {
	if len(args) > 0 {
		l.logger.Error().Fields(argsToFields(args...)).Msg(msg)
	} else {
		l.logger.Error().Msg(msg)
	}
}

func (l *MambaLogger) Fatal(msg string, args ...interface{}) {
	if len(args) > 0 {
		l.logger.Fatal().Fields(argsToFields(args...)).Msg(msg)
	} else {
		l.logger.Fatal().Msg(msg)
	}
}

func (l *MambaLogger) With(args ...interface{}) Logger {
	if len(args) > 0 {
		return &MambaLogger{
			logger: l.logger.With().Fields(argsToFields(args...)).Logger(),
			config: l.config,
			closer: l.closer,
		}
	}
	return l
}

func (l *MambaLogger) Sync() error {
	// If writing to a file, flush it safely to disk
	if f, ok := l.closer.(*os.File); ok && f != nil {
		return f.Sync()
	}
	return nil
}

func argsToFields(args ...interface{}) map[string]interface{} {
	fields := make(map[string]interface{})
	length := len(args)
	
	for i := 0; i < length; i += 2 {
		// Ensure we don't go out of bounds if an odd number of args are passed
		if i+1 < length {
			if key, ok := args[i].(string); ok {
				fields[key] = args[i+1]
			} else {
				// If key isn't a string, convert it to string dynamically
				fields[fmt.Sprintf("%v", args[i])] = args[i+1]
			}
		} else {
			// Catch an unpaired trailing argument
			fields["EXTRA_UNPAIRED_VALUE"] = args[i]
		}
	}
	return fields
}