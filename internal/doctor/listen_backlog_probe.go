package doctor

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"
)

const listenCounterSampleInterval = 250 * time.Millisecond

type ListenBacklogProbe struct {
	ReadCounters func() (TCPListenCounters, error)
	SampleWait   func(context.Context, time.Duration) error
}

func (p ListenBacklogProbe) Name() string { return "listen" }

func (p ListenBacklogProbe) Run(ctx context.Context) Result {
	start := time.Now()
	latency := func() int64 { return time.Since(start).Milliseconds() }

	if runtime.GOOS != "linux" {
		return Result{Name: "listen", Status: StatusSkip, Detail: "linux only", Latency: latency()}
	}

	somaxconn, err := readSysctlInt("/proc/sys/net/core/somaxconn")
	if err != nil {
		return Result{Name: "listen", Status: StatusFail, Detail: err.Error(), Latency: latency()}
	}

	read := p.ReadCounters
	if read == nil {
		read = ReadTCPListenCounters
	}
	wait := p.SampleWait
	if wait == nil {
		wait = waitContext
	}

	before, err := read()
	if err != nil {
		return Result{Name: "listen", Status: StatusFail, Detail: err.Error(), Latency: latency()}
	}
	if err := wait(ctx, listenCounterSampleInterval); err != nil {
		return Result{Name: "listen", Status: StatusFail, Detail: err.Error(), Latency: latency()}
	}
	after, err := read()
	if err != nil {
		return Result{Name: "listen", Status: StatusFail, Detail: err.Error(), Latency: latency()}
	}

	delta := before.Delta(after)
	var warns []string
	if somaxconn < 4096 {
		warns = append(warns, fmt.Sprintf("net.core.somaxconn=%d want >= 4096", somaxconn))
	}
	if delta.ListenOverflows > 0 {
		warns = append(warns, fmt.Sprintf("TcpExtListenOverflows delta=%d", delta.ListenOverflows))
	}
	if delta.ListenDrops > 0 {
		warns = append(warns, fmt.Sprintf("TcpExtListenDrops delta=%d", delta.ListenDrops))
	}
	if len(warns) > 0 {
		return Result{
			Name:    "listen",
			Status:  StatusWarn,
			Detail:  strings.Join(warns, "; "),
			Latency: latency(),
		}
	}

	return Result{
		Name:   "listen",
		Status: StatusPass,
		Detail: fmt.Sprintf("somaxconn=%d; ListenOverflows=%d ListenDrops=%d (no delta during probe)",
			somaxconn, after.ListenOverflows, after.ListenDrops),
		Latency: latency(),
	}
}

func waitContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
