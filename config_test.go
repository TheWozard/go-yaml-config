package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testConfig struct {
	Server    Server    `yaml:"server"`
	Tailscale Tailscale `yaml:"tailscale"`
	Logger    Logger    `yaml:"logger"`
	Name      string    `yaml:"name" default:"app"`
}

func TestLoad_MissingFileAppliesDefaults(t *testing.T) {
	cfg, err := Load[testConfig](filepath.Join(t.TempDir(), "missing.yaml"))
	require.NoError(t, err)

	require.Equal(t, "8080", cfg.Server.Port)
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

func TestMustLoad_MissingFileAppliesDefaults(t *testing.T) {
	cfg := MustLoad[testConfig](filepath.Join(t.TempDir(), "missing.yaml"))

	require.Equal(t, "8080", cfg.Server.Port)
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
