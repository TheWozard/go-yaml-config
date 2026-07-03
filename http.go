package config

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/TheWozard/go-yaml-config/log"
	"tailscale.com/tsnet"
)

type Tailscale struct {
	Hostname string `yaml:"hostname" env:"HOSTNAME"`
	Dir      string `yaml:"dir" env:"DIR" env-default:"/var/lib/tailscale"`
}

func (t Tailscale) Enabled() bool { return t.Hostname != "" }

// Listen starts serving handler over Tailscale if configured, otherwise
// over plain HTTP, and blocks until ctx is cancelled.
func (t Tailscale) Listen(ctx context.Context, handler http.Handler, logger *log.Logger, server Server) error {
	if t.Enabled() {
		return t.listen(ctx, handler, logger, server)
	}
	return server.Listen(ctx, handler, logger)
}

// listen starts an HTTPS server over Tailscale and blocks until ctx is
// cancelled, giving in-flight requests up to server.ShutdownTimeout to
// finish before the listener is forcibly closed.
func (t Tailscale) listen(ctx context.Context, handler http.Handler, logger *log.Logger, server Server) error {
	ts := &tsnet.Server{
		Dir:      t.Dir,
		Hostname: t.Hostname,
	}
	defer func() {
		if err := ts.Close(); err != nil {
			logger.WarnOfError("tailscale close", err)
		}
	}()

	ln, err := ts.ListenTLS("tcp", fmt.Sprintf(":%s", server.Port))
	if err != nil {
		return fmt.Errorf("tailscale listen: %w", err)
	}

	if lc, err := ts.LocalClient(); err == nil {
		if status, err := lc.Status(ctx); err == nil && status.Self != nil {
			logger.Info("listening", "addr", "https://"+status.Self.DNSName)
		}
	} else {
		logger.Info("listening", "addr", "https://"+t.Hostname)
	}

	return server.withDefaultName("tailscale").serve(ctx, ln, handler, logger)
}

type Server struct {
	Name            string        `yaml:"name" env:"NAME"`
	Port            string        `yaml:"port" env:"PORT" env-default:"8080"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"SHUTDOWN_TIMEOUT" env-default:"10s"`
}

// withDefaultName returns a copy of s with Name set to fallback if s.Name is
// empty.
func (s Server) withDefaultName(fallback string) Server {
	if s.Name == "" {
		s.Name = fallback
	}
	return s
}

// Listen starts an HTTP server and blocks until ctx is cancelled, giving
// in-flight requests up to ShutdownTimeout to finish before the listener is
// forcibly closed.
func (s Server) Listen(ctx context.Context, handler http.Handler, logger *log.Logger) error {
	addr := fmt.Sprintf(":%s", s.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	logger.Info("listening", "addr", "http://localhost"+addr)
	return s.withDefaultName("http").serve(ctx, ln, handler, logger)
}

// serve runs handler on ln until ctx is cancelled, giving in-flight requests
// up to s.ShutdownTimeout to finish before the listener is forcibly closed.
func (s Server) serve(ctx context.Context, ln net.Listener, handler http.Handler, logger *log.Logger) error {
	srv := &http.Server{Handler: handler}
	go s.gracefulShutdown(ctx, srv, logger)

	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// gracefulShutdown waits for ctx to be cancelled, then shuts srv down. If
// s.ShutdownTimeout elapses before in-flight requests finish, it forcibly
// closes the remaining connections rather than blocking forever.
func (s Server) gracefulShutdown(ctx context.Context, srv *http.Server, logger *log.Logger) {
	<-ctx.Done()
	logger.Info("shutting down", "server", s.Name)

	shutdownCtx := context.Background()
	if s.ShutdownTimeout > 0 {
		var cancel context.CancelFunc
		shutdownCtx, cancel = context.WithTimeout(shutdownCtx, s.ShutdownTimeout)
		defer cancel()
	}

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.WarnOfError("graceful shutdown timed out, forcing close", err, "server", s.Name)
		if closeErr := srv.Close(); closeErr != nil {
			logger.Error("force close", closeErr, "server", s.Name)
		}
	}
}
