package lifecycle

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"
)

const bodyOK = "OK"

func BodyOK() string { return bodyOK }

type Liveness struct {
	hits atomic.Uint64
}

func (l *Liveness) Hit() {
	if l != nil {
		l.hits.Add(1)
	}
}

func (l *Liveness) Hits() uint64 {
	if l == nil {
		return 0
	}
	return l.hits.Load()
}

func ServeHealthz(l *Liveness, w http.ResponseWriter, _ *http.Request) {
	if l != nil {
		l.Hit()
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(bodyOK))
}

type ReadinessProbe struct {
	ready atomic.Int32
}

func (p *ReadinessProbe) SetReady(ok bool) {
	if p == nil {
		return
	}
	if ok {
		p.ready.Store(1)
	} else {
		p.ready.Store(0)
	}
}

func (p *ReadinessProbe) Ready() bool {
	return p != nil && p.ready.Load() == 1
}

func (p *ReadinessProbe) ServeReadyz(w http.ResponseWriter, _ *http.Request) {
	if p.Ready() {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(bodyOK))
		return
	}
	http.Error(w, "not ready", http.StatusServiceUnavailable)
}

func (p *ReadinessProbe) StartBackground(ctx context.Context, interval time.Duration, check func(context.Context) bool) {
	if p == nil || check == nil {
		return
	}
	p.SetReady(true)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				probeCtx, cancel := context.WithTimeout(ctx, interval)
				p.SetReady(check(probeCtx))
				cancel()
			}
		}
	}()
}

func Register(mux *http.ServeMux, live *Liveness, ready *ReadinessProbe) {
	if mux == nil {
		return
	}
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		ServeHealthz(live, w, r)
	})
	if ready != nil {
		mux.HandleFunc("GET /readyz", ready.ServeReadyz)
	}
}
