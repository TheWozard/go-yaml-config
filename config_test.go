package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testConfig struct {
	Server    Server    `yaml:"server" env-prefix:"SERVER_"`
	Tailscale Tailscale `yaml:"tailscale" env-prefix:"TAILSCALE_"`
	Logger    Logger    `yaml:"logger" env-prefix:"LOGGER_"`
	Name      string    `yaml:"name" env:"APP_NAME" env-default:"app"`
	Debug     bool      `yaml:"debug"`
	Count     int       `yaml:"count"`
	Ratio     float64   `yaml:"ratio"`
}

func TestLoad_MissingFileAppliesDefaults(t *testing.T) {
	cfg, err := Load[testConfig](filepath.Join(t.TempDir(), "missing.yaml"))
	require.NoError(t, err)

	// Port has no config-level default: Server.Listen/Tailscale.listen apply
	// 80/443 internally at call time instead.
	require.Equal(t, "", cfg.Server.Port)
	require.Equal(t, 10*time.Second, cfg.Server.ShutdownTimeout)
	require.Equal(t, "/var/lib/tailscale", cfg.Tailscale.Dir)
	require.Equal(t, "app", cfg.Name)
}

func TestLoad_FileOverridesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
server:
  port: "9090"
name: custom
`), 0o644))

	cfg, err := Load[testConfig](path)
	require.NoError(t, err)

	require.Equal(t, "9090", cfg.Server.Port)
	require.Equal(t, 10*time.Second, cfg.Server.ShutdownTimeout)
	require.Equal(t, "custom", cfg.Name)
}

func TestLoad_MultipleFilesAppliedInOrder(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.yaml")
	override := filepath.Join(dir, "override.yaml")
	require.NoError(t, os.WriteFile(base, []byte(`
server:
  port: "9090"
name: base
`), 0o644))
	require.NoError(t, os.WriteFile(override, []byte(`
name: override
`), 0o644))

	cfg, err := Load[testConfig](base, override)
	require.NoError(t, err)

	require.Equal(t, "9090", cfg.Server.Port) // untouched by override, retained from base
	require.Equal(t, "override", cfg.Name)    // later path wins
}

func TestLoad_MissingIntermediateFileSkipped(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.yaml")
	require.NoError(t, os.WriteFile(base, []byte("name: base\n"), 0o644))

	cfg, err := Load[testConfig](base, filepath.Join(dir, "missing.yaml"))
	require.NoError(t, err)
	require.Equal(t, "base", cfg.Name)
}

func TestLoad_InvalidYAMLReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("not: [valid: yaml"), 0o644))

	_, err := Load[testConfig](path)
	require.Error(t, err)
}

func TestLoad_DurationStringParsedIntoField(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
server:
  shutdown_timeout: 1m30s
`), 0o644))

	cfg, err := Load[testConfig](path)
	require.NoError(t, err)
	require.Equal(t, 90*time.Second, cfg.Server.ShutdownTimeout)
}

func TestLoad_InvalidDurationStringReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
server:
  shutdown_timeout: not-a-duration
`), 0o644))

	_, err := Load[testConfig](path)
	require.Error(t, err)
}

func TestLoad_ScalarCoercedIntoStringField(t *testing.T) {
	cases := []struct {
		name string
		yaml string
		want string
	}{
		{"unquoted integer", "name: 123", "123"},
		{"unquoted boolean", "name: true", "true"},
		{"unquoted float", "name: 1.5", "1.5"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			require.NoError(t, os.WriteFile(path, []byte(tc.yaml), 0o644))

			cfg, err := Load[testConfig](path)
			require.NoError(t, err)
			require.Equal(t, tc.want, cfg.Name)
		})
	}
}

func TestLoad_BoolParsedIntoField(t *testing.T) {
	cases := []struct {
		name string
		yaml string
		want bool
	}{
		{"true", "debug: true", true},
		{"false", "debug: false", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			require.NoError(t, os.WriteFile(path, []byte(tc.yaml), 0o644))

			cfg, err := Load[testConfig](path)
			require.NoError(t, err)
			require.Equal(t, tc.want, cfg.Debug)
		})
	}
}

func TestLoad_InvalidBoolStringReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("debug: not-a-bool"), 0o644))

	_, err := Load[testConfig](path)
	require.Error(t, err)
}

func TestLoad_IntParsedIntoField(t *testing.T) {
	cases := []struct {
		name string
		yaml string
		want int
	}{
		{"positive", "count: 42", 42},
		{"negative", "count: -7", -7},
		{"zero", "count: 0", 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			require.NoError(t, os.WriteFile(path, []byte(tc.yaml), 0o644))

			cfg, err := Load[testConfig](path)
			require.NoError(t, err)
			require.Equal(t, tc.want, cfg.Count)
		})
	}
}

func TestLoad_InvalidIntStringReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("count: not-an-int"), 0o644))

	_, err := Load[testConfig](path)
	require.Error(t, err)
}

func TestLoad_FloatParsedIntoField(t *testing.T) {
	cases := []struct {
		name string
		yaml string
		want float64
	}{
		{"decimal", "ratio: 1.5", 1.5},
		{"integer literal", "ratio: 2", 2.0},
		{"negative", "ratio: -0.25", -0.25},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			require.NoError(t, os.WriteFile(path, []byte(tc.yaml), 0o644))

			cfg, err := Load[testConfig](path)
			require.NoError(t, err)
			require.Equal(t, tc.want, cfg.Ratio)
		})
	}
}

func TestLoad_InvalidFloatStringReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("ratio: not-a-float"), 0o644))

	_, err := Load[testConfig](path)
	require.Error(t, err)
}

func TestLoad_EnvVarOverridesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
server:
  port: "9090"
`), 0o644))
	t.Setenv("SERVER_PORT", "7070")

	cfg, err := Load[testConfig](path)
	require.NoError(t, err)
	require.Equal(t, "7070", cfg.Server.Port)
}

func TestLoad_EnvVarAppliesWithNoFile(t *testing.T) {
	t.Setenv("SERVER_PORT", "7070")

	cfg, err := Load[testConfig](filepath.Join(t.TempDir(), "missing.yaml"))
	require.NoError(t, err)
	require.Equal(t, "7070", cfg.Server.Port)
}

func TestLoad_FileValueRetainedWhenEnvVarUnset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
server:
  port: "9090"
`), 0o644))

	cfg, err := Load[testConfig](path)
	require.NoError(t, err)
	require.Equal(t, "9090", cfg.Server.Port)
}

func TestMustLoad_MissingFileAppliesDefaults(t *testing.T) {
	cfg := MustLoad[testConfig](filepath.Join(t.TempDir(), "missing.yaml"))

	require.Equal(t, "", cfg.Server.Port)
	require.Equal(t, "app", cfg.Name)
}

func TestMustLoad_InvalidYAMLPanics(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("not: [valid: yaml"), 0o644))

	require.Panics(t, func() {
		MustLoad[testConfig](path)
	})
}

func TestLoadEnv_ReadsPathFromEnvVar(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("name: from-env\n"), 0o644))
	t.Setenv("TEST_CONFIG_PATH", path)

	cfg, err := LoadEnv[testConfig]("TEST_CONFIG_PATH")
	require.NoError(t, err)
	require.Equal(t, "from-env", cfg.Name)
}

func TestLoadEnv_UnsetVarReturnsError(t *testing.T) {
	_, err := LoadEnv[testConfig]("TEST_CONFIG_PATH_UNSET")
	require.Error(t, err)
}

func TestLoadEnv_CommaSeparatedListAppliedInOrder(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.yaml")
	override := filepath.Join(dir, "override.yaml")
	require.NoError(t, os.WriteFile(base, []byte("name: base\n"), 0o644))
	require.NoError(t, os.WriteFile(override, []byte("name: override\n"), 0o644))
	t.Setenv("TEST_CONFIG_PATHS", base+", "+override)

	cfg, err := LoadEnv[testConfig]("TEST_CONFIG_PATHS")
	require.NoError(t, err)
	require.Equal(t, "override", cfg.Name)
}

func TestMustLoadEnv_ReadsPathFromEnvVar(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("name: from-env\n"), 0o644))
	t.Setenv("TEST_CONFIG_PATH", path)

	cfg := MustLoadEnv[testConfig]("TEST_CONFIG_PATH")
	require.Equal(t, "from-env", cfg.Name)
}

func TestMustLoadEnv_UnsetVarPanics(t *testing.T) {
	require.Panics(t, func() {
		MustLoadEnv[testConfig]("TEST_CONFIG_PATH_UNSET")
	})
}
