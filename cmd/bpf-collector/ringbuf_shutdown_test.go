package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/rlimit"
	"github.com/stretchr/testify/require"
)

func TestProbeRun_stop_closesRingbufBeforeWait(t *testing.T) {
	if err := rlimit.RemoveMemlock(); err != nil {
		t.Skipf("integration: memlock rlimit: %v", err)
	}

	m, err := ebpf.NewMap(&ebpf.MapSpec{
		Type:       ebpf.RingBuf,
		MaxEntries: 4096,
	})
	if err != nil {
		if errors.Is(err, ebpf.ErrNotSupported) {
			t.Skipf("integration: ringbuf map: %v", err)
		}
		t.Skipf("integration: ringbuf map (CAP_BPF/memlock): %v", err)
	}
	defer func() { _ = m.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := &probeRun{}
	r.ringWG.Add(1)
	go r.drainRingbuf(ctx, m)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		r.ringMu.Lock()
		ready := r.ringRD != nil
		r.ringMu.Unlock()
		if ready {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	r.ringMu.Lock()
	require.NotNil(t, r.ringRD, "drain goroutine should register ringbuf reader")
	r.ringMu.Unlock()

	done := make(chan struct{})
	go func() {
		cancel()
		r.closeRingbufReader()
		r.ringWG.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ringWG.Wait blocked: closeRingbufReader must run before Wait when Read has no events")
	}
}

func TestProbeRun_stop_cancelAloneBlocksUntilReaderClose_holdout(t *testing.T) {
	if err := rlimit.RemoveMemlock(); err != nil {
		t.Skipf("integration: memlock rlimit: %v", err)
	}

	m, err := ebpf.NewMap(&ebpf.MapSpec{
		Type:       ebpf.RingBuf,
		MaxEntries: 4096,
	})
	if err != nil {
		t.Skipf("integration: ringbuf map (CAP_BPF/memlock): %v", err)
	}
	defer func() { _ = m.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := &probeRun{}
	r.ringWG.Add(1)
	go r.drainRingbuf(ctx, m)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		r.ringMu.Lock()
		ready := r.ringRD != nil
		r.ringMu.Unlock()
		if ready {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	r.ringMu.Lock()
	require.NotNil(t, r.ringRD)
	r.ringMu.Unlock()

	cancel()
	waitDone := make(chan struct{})
	go func() {
		r.ringWG.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		t.Fatal("holdout: ringWG.Wait returned after cancel only; must close reader before Wait")
	case <-time.After(300 * time.Millisecond):
	}

	r.closeRingbufReader()
	select {
	case <-waitDone:
	case <-time.After(2 * time.Second):
		t.Fatal("ringWG.Wait still blocked after closeRingbufReader")
	}
}
