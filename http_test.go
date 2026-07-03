package config

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/TheWozard/go-yaml-config/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discardLogger() *log.Logger {
	return &log.Logger{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func TestTailscale_Enabled(t *testing.T) {
	assert.False(t, Tailscale{}.Enabled())
	assert.True(t, Tailscale{Hostname: "example"}.Enabled())
}

func TestServe_GracefulShutdown(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	release := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	ctx, cancel := context.WithCancel(context.Background())
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- Server{Name: "test", ShutdownTimeout: 2 * time.Second}.serve(ctx, ln, handler, discardLogger())
	}()

	reqDone := make(chan *http.Response, 1)
	reqErr := make(chan error, 1)
	go func() {
		resp, err := http.Get("http://" + ln.Addr().String())
		if err != nil {
			reqErr <- err
			return
		}
		reqDone <- resp
	}()

	time.Sleep(50 * time.Millisecond) // let the request reach the handler and block there

	cancel()       // trigger shutdown while the request is still in flight
	close(release) // let the handler finish so shutdown can complete

	select {
	case resp := <-reqDone:
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	case err := <-reqErr:
		t.Fatalf("in-flight request failed during graceful shutdown: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("in-flight request never completed")
	}

	select {
	case err := <-serveErr:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not return after shutdown")
	}
}

func TestServe_ForcedCloseOnTimeout(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	block := make(chan struct{}) // never closed: handler hangs forever
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block
	})

	ctx, cancel := context.WithCancel(context.Background())
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- Server{Name: "test", ShutdownTimeout: 30 * time.Millisecond}.serve(ctx, ln, handler, discardLogger())
	}()

	go func() {
		_, _ = http.Get("http://" + ln.Addr().String())
	}()

	time.Sleep(20 * time.Millisecond) // let the request reach the handler
	start := time.Now()
	cancel()

	select {
	case err := <-serveErr:
		assert.NoError(t, err)
		assert.Less(t, time.Since(start), time.Second, "shutdown timeout should force-close the blocked connection")
	case <-time.After(2 * time.Second):
		t.Fatal("serve should have force-closed the blocked connection instead of hanging")
	}
}
