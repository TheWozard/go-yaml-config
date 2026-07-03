package config

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// SignalContext returns a context that is cancelled when the process
// receives SIGINT or SIGTERM, along with a stop function that releases the
// underlying signal notification and should be deferred by the caller.
//
// The returned context is intended to be passed to Server.Listen or
// Tailscale.Listen, whose graceful shutdown triggers when the context is
// cancelled.
func SignalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}
