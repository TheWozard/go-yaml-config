//go:build !windows

package config

import (
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSignalContext_CancelsOnSIGTERM(t *testing.T) {
	ctx, stop := SignalContext()
	defer stop()

	require.NoError(t, syscall.Kill(os.Getpid(), syscall.SIGTERM))

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("context was not cancelled after SIGTERM")
	}
}

func TestSignalContext_StopCancelsContext(t *testing.T) {
	ctx, stop := SignalContext()
	stop()

	select {
	case <-ctx.Done():
	default:
		t.Fatal("context should be cancelled once stop is called")
	}
}
