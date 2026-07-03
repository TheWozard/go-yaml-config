package config

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger_New(t *testing.T) {
	cases := []struct {
		name  string
		level string
		want  slog.Level
	}{
		{"debug", "debug", slog.LevelDebug},
		{"info", "info", slog.LevelInfo},
		{"warn", "warn", slog.LevelWarn},
		{"error", "error", slog.LevelError},
		{"invalid level defaults to info", "not-a-level", slog.LevelInfo},
		{"empty level defaults to info", "", slog.LevelInfo},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := Logger{Level: tc.level}.New()
			ctx := context.Background()

			assert.True(t, l.Enabled(ctx, tc.want))
			if tc.want > slog.LevelDebug {
				assert.False(t, l.Enabled(ctx, tc.want-1))
			}
		})
	}
}

func TestLogger_New_Format(t *testing.T) {
	cases := []struct {
		name   string
		format string
		isJSON bool
	}{
		{"default is text", "", false},
		{"text", "text", false},
		{"json", "json", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := Logger{Format: tc.format}.New()

			_, ok := l.Handler().(*slog.JSONHandler)
			assert.Equal(t, tc.isJSON, ok)
		})
	}
}
