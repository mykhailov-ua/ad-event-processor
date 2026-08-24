package server

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"ad-event-processor/pkg/broker/client"
)

func TestFetchHighWatermark(t *testing.T) {
	dir, err := os.MkdirTemp("", "fetch-hwm-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	s := NewServer("127.0.0.1:0", dir, 10*1024*1024, 4096)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	time.Sleep(150 * time.Millisecond)

	topic := "hwm-topic"
	cli := client.NewClient(s.Addr(), 2*time.Second)
	if err := cli.Connect(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cli.Close() }()

	const n = 5
	for i := range n {
		if _, err := cli.Produce(context.Background(), topic, 0, []byte(fmt.Sprintf("m-%d", i))); err != nil {
			t.Fatal(err)
		}
	}

	iter, err := cli.Fetch(context.Background(), topic, 0, 0, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	if iter.HighWatermark != n {
		t.Fatalf("expected hwm %d after produce, got %d", n, iter.HighWatermark)
	}

	empty, err := cli.Fetch(context.Background(), topic, 0, n, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if empty.HighWatermark != n {
		t.Fatalf("expected hwm %d on empty tail fetch, got %d", n, empty.HighWatermark)
	}
	if empty.Next() {
		t.Fatal("expected no messages at tail offset")
	}
}

func TestFault_MonotonicReads_HighWatermarkNeverRegresses(t *testing.T) {
	dir, err := os.MkdirTemp("", "monotonic-hwm-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	s := NewServer("127.0.0.1:0", dir, 10*1024*1024, 4096)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	time.Sleep(150 * time.Millisecond)

	topic := "mono-topic"
	cli := client.NewClient(s.Addr(), 2*time.Second)
	if err := cli.Connect(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cli.Close() }()

	var lastHWM uint64
	for i := range 10 {
		if _, err := cli.Produce(context.Background(), topic, 0, []byte(fmt.Sprintf("m-%d", i))); err != nil {
			t.Fatal(err)
		}
		iter, err := cli.Fetch(context.Background(), topic, 0, 0, 4096)
		if err != nil {
			t.Fatal(err)
		}
		if iter.HighWatermark < lastHWM {
			t.Fatalf("hwm regressed: was %d now %d at produce %d", lastHWM, iter.HighWatermark, i)
		}
		if iter.HighWatermark != uint64(i+1) {
			t.Fatalf("expected hwm %d, got %d", i+1, iter.HighWatermark)
		}
		lastHWM = iter.HighWatermark
	}

	t.Logf("fault_proof fault=monotonic_reads hwm_monotonic=true final_hwm=%d", lastHWM)
}
