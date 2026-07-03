// Package config provides small, opinionated building blocks for loading
// application configuration from YAML files, environment variables, and
// defaults, plus a couple of ready-made pieces (HTTP server, Tailscale
// listener, slog logger) that most services need.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
)

// Load reads YAML files at paths into a new T, applied in order so later
// paths override earlier ones, then applies environment variable overrides
// from `env:"..."` tags. Fields tagged `env-default:"..."` are used when
// nothing else set the field. Precedence, lowest to highest: env-default,
// YAML files (in order), env vars. A path that does not exist is skipped
// without error.
func Load[T any](paths ...string) (T, error) {
	var dst T

	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return dst, err
		}
		if err := cleanenv.ReadConfig(path, &dst); err != nil {
			return dst, err
		}
	}

	// Guarantees env-default/env overrides apply even when every path was
	// skipped; otherwise a harmless repeat of what ReadConfig already did.
	if err := cleanenv.ReadEnv(&dst); err != nil {
		return dst, err
	}

	return dst, nil
}

// MustLoad is like Load but panics if loading fails.
func MustLoad[T any](paths ...string) T {
	dst, err := Load[T](paths...)
	if err != nil {
		panic(err)
	}
	return dst
}

// LoadEnv is like Load, but reads file paths from the named environment
// variable rather than taking them directly, as a comma-separated list
// applied in order so later paths override earlier ones. It errors if the
// variable is not set.
func LoadEnv[T any](envVar string) (T, error) {
	value, ok := os.LookupEnv(envVar)
	if !ok {
		var dst T
		return dst, fmt.Errorf("environment variable %q not set", envVar)
	}

	paths := strings.Split(value, ",")
	for i, path := range paths {
		paths[i] = strings.TrimSpace(path)
	}

	return Load[T](paths...)
}

// MustLoadEnv is like LoadEnv but panics if loading fails.
func MustLoadEnv[T any](envVar string) T {
	dst, err := LoadEnv[T](envVar)
	if err != nil {
		panic(err)
	}
	return dst
}
