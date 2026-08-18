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

func (s *redisStub) SMembers(_ context.Context, key string) *redis.StringSliceCmd {
	cmd := redis.NewStringSliceCmd(context.Background())
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
	t.Cleanup(func() { m.Close() })
	return m
}
