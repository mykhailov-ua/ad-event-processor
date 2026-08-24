package lifecycle

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ad-event-processor/internal/config"
)

type Timeouts struct {
	Shutdown time.Duration
	Wait     time.Duration
}

func TimeoutsFromConfig(cfg *config.Config) Timeouts {
	return Timeouts{
		Shutdown: time.Duration(cfg.Lifecycle.ShutdownTimeoutMs) * time.Millisecond,
		Wait:     time.Duration(cfg.Lifecycle.WaitTimeoutMs) * time.Millisecond,
	}
}

func TimeoutsFromEnv() Timeouts {
	return Timeouts{
		Shutdown: config.LifecycleShutdownTimeout(),
		Wait:     config.LifecycleWaitTimeout(),
	}
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
