# go-yaml-config

Small, opinionated building blocks for loading app configuration from YAML,
with sane defaults and a couple of ready-made pieces (HTTP server, Tailscale
listener, slog logger) that most services need.

## Install

```sh
go get github.com/TheWozard/go-yaml-config
```

## Loading config

`Load[T]` fills `T` in three layers, lowest precedence first: `env-default:"..."`
tags, then any number of YAML files overlaid on top (applied in order so later
files override earlier ones), then `env:"..."` tags read from the environment,
which win over everything else. A missing file is skipped, not an error — you
just get whatever was already set. `MustLoad[T]` is the same but panics instead
of returning an error, for use during startup where a bad config should be
fatal.

```go
cfg := config.MustLoad[AppConfig]("base.yaml", "local.yaml")
```

`LoadEnv[T]` and `MustLoadEnv[T]` are the same as `Load`/`MustLoad`, except
they take the name of a single environment variable holding a
comma-separated list of config paths rather than the paths themselves,
again applied in order. Unlike a missing file, an unset environment variable
is an error, since the caller has said the paths should come from there.

```go
// APP_CONFIG_PATHS=base.yaml,local.yaml
cfg := config.MustLoadEnv[AppConfig]("APP_CONFIG_PATHS")
```

```go
package main

import (
	"net/http"

	"github.com/TheWozard/go-yaml-config"
)

type AppConfig struct {
	Name      string           `yaml:"name" env:"APP_NAME" env-default:"app"`
	Server    config.Server    `yaml:"server" env-prefix:"SERVER_"`
	Tailscale config.Tailscale `yaml:"tailscale" env-prefix:"TAILSCALE_"`
	Logger    config.Logger    `yaml:"logger" env-prefix:"LOGGER_"`
}

func main() {
	cfg := config.MustLoad[AppConfig]("config.yaml")

	logger := cfg.Logger.New()
	logger.Info("starting", "name", cfg.Name)

	ctx, stop := config.SignalContext()
	defer stop()

	handler := http.NewServeMux()
	if err := cfg.Tailscale.Listen(ctx, handler, logger, cfg.Server); err != nil {
		logger.Error("server exited", err)
	}
}
```

```yaml
name: my-service
server:
  port: "9090"
  shutdown_timeout: 15s
logger:
  level: info
  format: json
```

`env:"..."` tags (with an `env-prefix:"..."` on the nested struct fields to
namespace them) let environment variables override the file for the same
`AppConfig` above — e.g. `SERVER_PORT=9091` wins over the YAML file's
`server.port: "9090"`, and `APP_NAME` wins over `name`.

## Pieces

### `SignalContext`

`SignalContext()` returns a `context.Context` that's cancelled on SIGINT or
SIGTERM, plus a `stop` function to release the signal handler (defer it).
Pass the context to `Server.Listen`/`Tailscale.Listen` so an OS shutdown
signal triggers their graceful shutdown.

### `Server`

HTTP server config (`name`/`NAME`, `port`/`PORT`, `shutdown_timeout`/
`SHUTDOWN_TIMEOUT`). `Listen` blocks until the context is cancelled, then
gives in-flight requests up to `shutdown_timeout` to finish before
force-closing. `name` is used in log lines to identify the server; if left
unset it defaults to `http`, or `tailscale` when served via
`Tailscale.Listen`.

### `Tailscale`

Optional Tailscale (`tsnet`) listener config (`hostname`/`HOSTNAME`,
`dir`/`DIR`). `Enabled()` reports whether a hostname was set.
`Tailscale.Listen` serves over Tailscale when enabled, falling back to
`Server.Listen` (plain HTTP) otherwise.

### `Logger`

```yaml
logger:
  level: info   # debug, info, warn, error (default: info)
  format: text  # text, json (default: text)
```

Both fields also read from the environment (`LEVEL`, `FORMAT`, or namespaced
per `env-prefix` as in the `AppConfig` example above). `Logger.New()` returns
a `*log.Logger` (a thin wrapper around `*slog.Logger`) writing to stderr.
`format: json` switches to `slog.NewJSONHandler`; anything else (including an
invalid `level`) falls back to the text default.

The wrapper adds `Error` and `WarnOfError`, which log at Error/Warn level
with the error attached under the `err` key:

```go
logger.Error("failed to connect", err, "addr", addr)
// level=ERROR msg="failed to connect" err="connection refused" addr=...
```

## Testing

```sh
go test ./...
```

`make check` runs everything CI does — build, vet, lint (`golangci-lint`),
test (with `-race`), and `govulncheck` — via `go run`, so no tools need to be
installed beforehand.
