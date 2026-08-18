package doctor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/bidshard/ad-event-processor/internal/edge"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEdgeXDPProbe_skipsWhenDisabled(t *testing.T) {
	probe := EdgeXDPProbe{ConfigEnabled: false}
	res := probe.Run(context.Background())
	assert.Equal(t, StatusSkip, res.Status)
}

func TestEdgeXDPProbe_failsWhenMapsAndUnitsMissing(t *testing.T) {
	dir := t.TempDir()
	btfPathForProbe = filepath.Join(dir, "vmlinux")
	blocklistMapPathForProbe = filepath.Join(dir, "missing-map")
	require.NoError(t, os.WriteFile(btfPathForProbe, []byte("btf"), 0o644))
	t.Cleanup(func() {
		btfPathForProbe = "/sys/kernel/btf/vmlinux"
		blocklistMapPathForProbe = "/sys/fs/bpf/ad-event-processor/blocklist_v4"
	})

	systemdUnitActiveFn = func(unit string) (bool, error) { return false, nil }
	t.Cleanup(func() { systemdUnitActiveFn = systemdUnitActive })

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	mr.HSet("entitlement:deployment", "ebpf_xdp_edge", "1")

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	probe := EdgeXDPProbe{
		ConfigEnabled: true,
		Deps: ProbeDeps{
			Redis: func(context.Context) ([]redis.UniversalClient, error) {
				return []redis.UniversalClient{rdb}, nil
			},
		},
	}
	res := probe.Run(context.Background())
	assert.Equal(t, StatusFail, res.Status)
	assert.Contains(t, res.Detail, "blocklist_v4 map not pinned")
	assert.Contains(t, res.Detail, "not active")
}

func TestEdgeXDPProbe_passesWhenReady(t *testing.T) {
	dir := t.TempDir()
	btfPathForProbe = filepath.Join(dir, "vmlinux")
	blocklistMapPathForProbe = filepath.Join(dir, "blocklist_v4")
	require.NoError(t, os.WriteFile(btfPathForProbe, []byte("btf"), 0o644))
	require.NoError(t, os.WriteFile(blocklistMapPathForProbe, []byte("map"), 0o644))
	t.Cleanup(func() {
		btfPathForProbe = "/sys/kernel/btf/vmlinux"
		blocklistMapPathForProbe = "/sys/fs/bpf/ad-event-processor/blocklist_v4"
	})

	systemdUnitActiveFn = func(unit string) (bool, error) { return true, nil }
	t.Cleanup(func() { systemdUnitActiveFn = systemdUnitActive })

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	mr.HSet("entitlement:deployment", "ebpf_xdp_edge", "1")

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	probe := EdgeXDPProbe{
		ConfigEnabled: true,
		Deps: ProbeDeps{
			Redis: func(context.Context) ([]redis.UniversalClient, error) {
				return []redis.UniversalClient{rdb}, nil
			},
		},
		StatsReader: func(context.Context) (edge.Snapshot, error) {
			return edge.Snapshot{UpdatedAt: time.Now().UTC()}, nil
		},
	}
	res := probe.Run(context.Background())
	assert.Equal(t, StatusPass, res.Status)
}
