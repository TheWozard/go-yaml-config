package config

import (
	"log/slog"
	"os"

	"github.com/TheWozard/go-yaml-config/log"
)

type Logger struct {
	Level  string `yaml:"level"`                // debug, info, warn, error (default: info)
	Format string `yaml:"format" default:"text"` // text, json (default: text)
}

// New creates a slog.Logger writing to stderr at the configured level, in
// either human-readable text (default) or JSON.
func (l Logger) New() *log.Logger {
	var level slog.Level
	if err := level.UnmarshalText([]byte(l.Level)); err != nil {
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if l.Format == "json" {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	return &log.Logger{
		Logger: slog.New(handler),
	}
}
