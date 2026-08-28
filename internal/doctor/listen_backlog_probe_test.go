package doctor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListenBacklogProbe_pass(t *testing.T) {
	reads := []TCPListenCounters{
		{ListenOverflows: 10, ListenDrops: 2},
		{ListenOverflows: 10, ListenDrops: 2},
	}
	idx := 0
	probe := ListenBacklogProbe{
		ReadCounters: func() (TCPListenCounters, error) {
			c := reads[idx]
			idx++
			return c, nil
		},
		SampleWait: func(context.Context, time.Duration) error { return nil },
	}

	result := probe.Run(context.Background())
	assert.Equal(t, StatusPass, result.Status)
}

func TestListenBacklogProbe_warnOnDelta(t *testing.T) {
	reads := []TCPListenCounters{
		{ListenOverflows: 10, ListenDrops: 2},
		{ListenOverflows: 12, ListenDrops: 5},
	}
	idx := 0
	probe := ListenBacklogProbe{
		ReadCounters: func() (TCPListenCounters, error) {
			c := reads[idx]
			idx++
			return c, nil
		},
		SampleWait: func(context.Context, time.Duration) error { return nil },
	}

	result := probe.Run(context.Background())
	assert.Equal(t, StatusWarn, result.Status)
	assert.Contains(t, result.Detail, "TcpExtListenOverflows delta=2")
	assert.Contains(t, result.Detail, "TcpExtListenDrops delta=3")
}

func TestRunOnlyListenFilter(t *testing.T) {
	probes := []Probe{
		stubProbe{name: "redis", result: Result{Name: "redis", Status: StatusPass}},
		ListenBacklogProbe{},
	}
	rep := Run(context.Background(), Options{
		Only:   []string{"listen"},
		Probes: probes,
	})
	require.Len(t, rep.Results, 1)
	assert.Equal(t, "listen", rep.Results[0].Name)
}

func TestCheckHint_listen(t *testing.T) {
	hint := CheckHint("listen")
	require.Contains(t, hint, "somaxconn")
}
