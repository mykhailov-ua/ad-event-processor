package edge

import (
	"context"
	"testing"

	"github.com/cilium/ebpf"
	"github.com/redis/go-redis/v9"
)

type redisStub struct {
	sets map[string][]string
	err  error
}

func (s *redisStub) SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	cmd := redis.NewStringSliceCmd(ctx)
	if s.err != nil {
		cmd.SetErr(s.err)
		return cmd
	}
	cmd.SetVal(append([]string(nil), s.sets[key]...))
	return cmd
}

func newLPMMap(t *testing.T) *ebpf.Map {
	t.Helper()
	m, err := ebpf.NewMap(&ebpf.MapSpec{
		Type:       ebpf.LPMTrie,
		KeySize:    8,
		ValueSize:  1,
		MaxEntries: 4096,
		Flags:      1,
	})
	if err != nil {
		t.Skipf("BPF map unavailable: %v", err)
	}
	t.Cleanup(func() { _ = m.Close() })
	return m
}

func newLPMMapBench(b *testing.B) *ebpf.Map {
	b.Helper()
	m, err := ebpf.NewMap(&ebpf.MapSpec{
		Type:       ebpf.LPMTrie,
		KeySize:    8,
		ValueSize:  1,
		MaxEntries: 4096,
		Flags:      1,
	})
	if err != nil {
		b.Skipf("BPF map unavailable: %v", err)
	}
	b.Cleanup(func() { _ = m.Close() })
	return m
}

func newLPMMapV6(t *testing.T) *ebpf.Map {
	t.Helper()
	m, err := ebpf.NewMap(&ebpf.MapSpec{
		Type:       ebpf.LPMTrie,
		KeySize:    20,
		ValueSize:  1,
		MaxEntries: 4096,
		Flags:      1,
	})
	if err != nil {
		t.Skipf("BPF map unavailable: %v", err)
	}
	t.Cleanup(func() { _ = m.Close() })
	return m
}
