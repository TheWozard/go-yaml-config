package config

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"os"

	"github.com/TheWozard/go-yaml-config/log"
	"gopkg.in/yaml.v3"
)

// Config holds the base configuration shared by all services.
// Embed this struct in your service-specific config to inherit Load and Listen behaviour.
//
//	type MyConfig struct {
//	    config.Config `yaml:",inline"`
//	    MyField string `yaml:"my_field"`
//	}
type Config struct {
	Tailscale Tailscale `yaml:"tailscale"`
	Server    Server    `yaml:"server"`
	Logger    Logger    `yaml:"logger"`
}

func defaultConfig() Config {
	return Config{
		Tailscale: Tailscale{
			Dir:  "/var/lib/tailscale",
			Port: "443",
		},
		Server: Server{
			Port: "8080",
		},
		Logger: Logger{Level: "info"},
	}
}

// Load reads a YAML file at path into dst, applying defaults and env overrides.
// dst must be a non-nil pointer. If the file does not exist the defaults are
// returned without error.
func Load[T any](path string, dst *T, defaults func(*T)) error {
	if defaults != nil {
		defaults(dst)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}

	return yaml.Unmarshal(data, dst)
}

// LoadBase reads a YAML file into a *Config with built-in defaults and env
// overrides applied. Use Load for service-specific config structs.
func LoadBase(path string) (*Config, error) {
	cfg := defaultConfig()
	if err := Load(path, &cfg, nil); err != nil {
		return nil, err
	}
	cfg.ApplyEnvOverrides()
	return &cfg, nil
}

// ApplyEnvOverrides lets PORT and TS_HOSTNAME take precedence over YAML values.
func (c *Config) ApplyEnvOverrides() {
	if v := os.Getenv("PORT"); v != "" {
		c.Server.Port = v
		c.Tailscale.Port = v
	}
	if v := os.Getenv("TS_HOSTNAME"); v != "" {
		c.Tailscale.Hostname = v
	}
}

// ServerListener is implemented by both Server (plain HTTP) and Tailscale (HTTPS).
type ServerListener interface {
	Listen(context.Context, http.Handler, *log.Logger) error
}

// GetServerListener returns Tailscale if configured, otherwise the plain HTTP server.
func (c *Config) GetServerListener() ServerListener {
	if c.Tailscale.Enabled() {
		return c.Tailscale
	}
	return c.Server
}
