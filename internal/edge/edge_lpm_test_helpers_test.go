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

func newHostHashMapBench(b *testing.B) *ebpf.Map {
	b.Helper()
	m, err := ebpf.NewMap(&ebpf.MapSpec{
		Type:       ebpf.LRUHash,
		KeySize:    4,
		ValueSize:  1,
		MaxEntries: 4096,
	})
	if err != nil {
		b.Skipf("BPF map unavailable: %v", err)
	}
	b.Cleanup(func() { _ = m.Close() })
	return m
}

func newTestBlocklistMapsV4OnlyBench(b *testing.B) BlocklistMaps {
	b.Helper()
	return BlocklistMaps{
		V4Host:   newHostHashMapBench(b),
		V4Prefix: newLPMMapBench(b),
	}
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

func newHostHashMapV4(t *testing.T) *ebpf.Map {
	t.Helper()
	m, err := ebpf.NewMap(&ebpf.MapSpec{
		Type:       ebpf.LRUHash,
		KeySize:    4,
		ValueSize:  1,
		MaxEntries: 4096,
	})
	if err != nil {
		t.Skipf("BPF map unavailable: %v", err)
	}
	t.Cleanup(func() { _ = m.Close() })
	return m
}

func newHostHashMapV6(t *testing.T) *ebpf.Map {
	t.Helper()
	m, err := ebpf.NewMap(&ebpf.MapSpec{
		Type:       ebpf.LRUHash,
		KeySize:    16,
		ValueSize:  1,
		MaxEntries: 4096,
	})
	if err != nil {
		t.Skipf("BPF map unavailable: %v", err)
	}
	t.Cleanup(func() { _ = m.Close() })
	return m
}

func newTestBlocklistMapsV4Only(t *testing.T) BlocklistMaps {
	t.Helper()
	return BlocklistMaps{
		V4Host:   newHostHashMapV4(t),
		V4Prefix: newLPMMap(t),
	}
}

func blocklistMapsFromObjects(objs *EdgeObjects) BlocklistMaps {
	return BlocklistMaps{
		V4Host:   objs.BlocklistHostV4,
		V4Prefix: objs.BlocklistV4,
		V6Host:   objs.BlocklistHostV6,
		V6Prefix: objs.BlocklistV6,
	}
}
