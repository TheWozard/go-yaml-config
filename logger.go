package config

import (
	"log/slog"
	"os"

	"github.com/TheWozard/go-yaml-config/log"
)

type Logger struct {
	Level string `yaml:"level"` // debug, info, warn, error (default: info)
}

// New creates a slog.Logger writing text to stderr at the configured level.
func (l Logger) New() *log.Logger {
	var level slog.Level
	if err := level.UnmarshalText([]byte(l.Level)); err != nil {
		level = slog.LevelInfo
	}
	return &log.Logger{
		Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})),
	}
}
