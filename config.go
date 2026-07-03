package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/creasty/defaults"
	"gopkg.in/yaml.v3"
)

// Load reads YAML files at paths into a new T, applied in order so later
// paths override earlier ones. Fields tagged `default:"..."` are set first
// so the files' contents can override them. A path that does not exist is
// skipped without error.
func Load[T any](paths ...string) (T, error) {
	var dst T

	if err := defaults.Set(&dst); err != nil {
		return dst, err
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return dst, err
		}
		if err := yaml.Unmarshal(data, &dst); err != nil {
			return dst, err
		}
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
