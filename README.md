# go-yaml-config

Small, opinionated building blocks for loading app configuration from YAML,
with sane defaults and a couple of ready-made pieces (HTTP server, Tailscale
listener, slog logger) that most services need.

## Install

```sh
go get github.com/TheWozard/go-yaml-config
```

## Loading config

`Load[T]` fills `T` with its `default:"..."` tags, then overlays any number
of YAML files on top, applied in order so later files override earlier
ones. A missing file is skipped, not an error — you just get whatever was
already set. `MustLoad[T]` is the same but panics instead of returning an
error, for use during startup where a bad config should be fatal.

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
	Name      string           `yaml:"name" default:"app"`
	Server    config.Server    `yaml:"server"`
	Tailscale config.Tailscale `yaml:"tailscale"`
	Logger    config.Logger    `yaml:"logger"`
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

## Pieces

### `SignalContext`

`SignalContext()` returns a `context.Context` that's cancelled on SIGINT or
SIGTERM, plus a `stop` function to release the signal handler (defer it).
Pass the context to `Server.Listen`/`Tailscale.Listen` so an OS shutdown
signal triggers their graceful shutdown.

### `Server`

HTTP server config (`name`, `port`, `shutdown_timeout`). `Listen` blocks
until the context is cancelled, then gives in-flight requests up to
`shutdown_timeout` to finish before force-closing. `name` is used in log
lines to identify the server; if left unset it defaults to `http`, or
`tailscale` when served via `Tailscale.Listen`.

### `Tailscale`

Optional Tailscale (`tsnet`) listener config (`hostname`, `dir`).
`Enabled()` reports whether a hostname was set. `Tailscale.Listen` serves
over Tailscale when enabled, falling back to `Server.Listen` (plain HTTP)
otherwise.

### `Logger`

```yaml
logger:
  level: info   # debug, info, warn, error (default: info)
  format: text  # text, json (default: text)
```

`Logger.New()` returns a `*log.Logger` (a thin wrapper around `*slog.Logger`)
writing to stderr. `format: json` switches to `slog.NewJSONHandler`; anything
else (including an invalid `level`) falls back to the text default.

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
