package lifecycle

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Timeouts struct {
	Shutdown time.Duration
	Wait     time.Duration
}

func TimeoutsFromMillis(shutdownMs, waitMs int) Timeouts {
	if shutdownMs <= 0 {
		shutdownMs = 15000
	}
	if waitMs <= 0 {
		waitMs = 5000
	}
	return Timeouts{
		Shutdown: time.Duration(shutdownMs) * time.Millisecond,
		Wait:     time.Duration(waitMs) * time.Millisecond,
	}
}

func TimeoutsFromEnv() Timeouts {
	return Timeouts{
		Shutdown: time.Duration(envInt("SHUTDOWN_TIMEOUT_MS", 15000)) * time.Millisecond,
		Wait:     time.Duration(envInt("WAIT_TIMEOUT_MS", 5000)) * time.Millisecond,
	}
}

func envInt(name string, def int) int {
	s := strings.TrimSpace(os.Getenv(name))
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return def
	}
	return n
}

func NotifyContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
}

func WaitSignal() os.Signal {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	sig := <-stop
	signal.Stop(stop)
	return sig
}

func ShutdownHTTPServer(srv *http.Server, timeout time.Duration) error {
	if srv == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return srv.Shutdown(ctx)
}

func Wait(timeout time.Duration, fn func()) error {
	if fn == nil {
		return nil
	}
	done := make(chan struct{})
	go func() {
		fn()
		close(done)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
