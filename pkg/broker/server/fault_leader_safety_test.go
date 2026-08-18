package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/broker/client"
	"github.com/bidshard/ad-event-processor/pkg/faultproof"
)

func newLeaderSafetyServer(t *testing.T) *Server {
	t.Helper()
	s := NewServer("127.0.0.1:0", t.TempDir(), 10*1024*1024, 4096)
	if err := s.Start(); err != nil {
		t.Fatalf("server start: %v", err)
	}
	t.Cleanup(s.Stop)
	return s
}

func partitionNextOffset(t *testing.T, s *Server, tpKey string) uint64 {
	t.Helper()
	pl, err := s.getOrCreatePartition(tpKey)
	if err != nil {
		t.Fatalf("partition %s: %v", tpKey, err)
	}
	return pl.NextOffset()
}

func TestFault_CoordinatorUnreachable_ProduceFailsClosed(t *testing.T) {
	s := newLeaderSafetyServer(t)

	deadRedis := allocFreeTCPAddr(t)
	coord, err := NewCoordinator("broker-unreachable", s.Addr(), "redis://"+deadRedis+"/0", s)
	if err != nil {
		t.Fatalf("coordinator: %v", err)
	}
	s.SetCoordinator(coord)
	coord.Start(context.Background())
	defer coord.Stop()

	topic := "leaderless-topic"
	tpKey := topicPartitionKey(topic)

	cli := client.NewClient(s.Addr(), 300*time.Millisecond)
	if err := cli.Connect(); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = cli.Close() }()

	if _, err := cli.Produce(context.Background(), topic, 0, []byte("must-not-persist")); err == nil {
		t.Fatal("produce accepted while no lease is held")
	}

	if got := partitionNextOffset(t, s, tpKey); got != 0 {
		t.Fatalf("log advanced without leadership: next offset %d", got)
	}
	if coord.IsLeader(tpKey) {
		t.Fatal("node claims leadership with unreachable coordinator")
	}

	faultproof.Log(t, "broker_leaderless_produce_fails_closed", map[string]string{
		"produce_rejected": "true",
		"log_offset":       "0",
		"leader":           "false",
	})
}

func TestFault_LeaseExpiry_SelfFencesProduce(t *testing.T) {
	redisURL, redisCleanup := startFaultRedis(t)
	redisAlive := true
	defer func() {
		if redisAlive {
			redisCleanup()
		}
	}()

	s := newLeaderSafetyServer(t)

	cfg := CoordConfig{
		LeaseTTL:           2 * time.Second,
		Interval:           200 * time.Millisecond,
		RenewFailThreshold: 2,
		DebounceWindow:     100 * time.Millisecond,
	}
	coord, err := NewCoordinatorWithConfig("broker-lease", s.Addr(), redisURL, s, cfg)
	if err != nil {
		t.Fatalf("coordinator: %v", err)
	}
	s.SetCoordinator(coord)
	coord.Start(context.Background())
	defer coord.Stop()

	topic := "lease-expiry-topic"
	tpKey := topicPartitionKey(topic)

	cli := client.NewClient(s.Addr(), 2*time.Second)
	if err := cli.Connect(); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = cli.Close() }()

	if _, err := cli.Produce(context.Background(), topic, 0, []byte("leader-write")); err != nil {
		t.Fatalf("produce under healthy lease failed: %v", err)
	}
	acceptedOffset := partitionNextOffset(t, s, tpKey)
	if acceptedOffset != 1 {
		t.Fatalf("expected one committed record, got next offset %d", acceptedOffset)
	}

	redisCleanup()
	redisAlive = false

	requireEventually(t, func() bool {
		return !coord.IsLeader(tpKey)
	}, 15*time.Second, 200*time.Millisecond, "node must self-fence after lease expiry")

	if _, err := cli.Produce(context.Background(), topic, 0, []byte("post-fence-write")); err == nil {
		t.Fatal("produce accepted after lease expiry")
	}
	if got := partitionNextOffset(t, s, tpKey); got != acceptedOffset {
		t.Fatalf("log advanced after self-fencing: %d -> %d", acceptedOffset, got)
	}

	faultproof.Log(t, "broker_lease_expiry_self_fence", map[string]string{
		"lease_ttl_ms":      "2000",
		"accepted_offset":   fmt.Sprintf("%d", acceptedOffset),
		"post_fence_reject": "true",
		"log_offset_stable": "true",
	})
}

func TestFault_LeaderTakeover_HWMNeverRegresses(t *testing.T) {
	redisURL, redisCleanup := startFaultRedis(t)
	defer redisCleanup()

	s := newLeaderSafetyServer(t)

	coord, err := NewCoordinator("broker-hwm", s.Addr(), redisURL, s)
	if err != nil {
		t.Fatalf("coordinator: %v", err)
	}
	s.SetCoordinator(coord)
	defer coord.Stop()

	topic := "hwm-monotonic-topic"
	tpKey := topicPartitionKey(topic)

	setHWM := func(value string) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := coord.rdb.Set(ctx, logHWMKey(tpKey), value, 0).Err(); err != nil {
			t.Fatalf("set hwm %s: %v", value, err)
		}
	}
	currentHWM := func() uint64 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return coord.readLogHWM(ctx, tpKey)
	}

	setHWM("100")

	coord.PublishLogHWM(context.Background(), tpKey, 50)
	if got := currentHWM(); got != 100 {
		t.Fatalf("hwm regressed on lagging publish: got %d want 100", got)
	}

	coord.PublishLogHWM(context.Background(), tpKey, 150)
	if got := currentHWM(); got != 150 {
		t.Fatalf("hwm did not advance: got %d want 150", got)
	}

	setHWM("100")
	pl, err := s.getOrCreatePartition(tpKey)
	if err != nil {
		t.Fatalf("partition: %v", err)
	}
	if _, err := pl.Append([]byte("local-0")); err != nil {
		t.Fatalf("seed local record: %v", err)
	}

	coord.setLeaderState(tpKey, true, 1, false)
	if coord.IsLeaderReady(tpKey) {
		t.Fatal("lagging leader must not be ready before catch-up")
	}

	coord.recoverLeaderReadiness(context.Background(), tpKey, 100, time.Now())

	if got := currentHWM(); got != 100 {
		t.Fatalf("catch-up timeout truncated cluster hwm: got %d want 100", got)
	}
	if !coord.IsLeaderReady(tpKey) {
		t.Fatal("leader must accept traffic after catch-up budget")
	}

	faultproof.Log(t, "broker_takeover_hwm_monotonic", map[string]string{
		"lagging_publish_ignored": "true",
		"hwm_after_timeout":       "100",
		"local_offset":            fmt.Sprintf("%d", pl.NextOffset()),
		"ready_with_gap":          "true",
	})
}
