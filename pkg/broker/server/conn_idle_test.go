package server

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/broker/client"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestPartialFrame_ReadIdleClosesSlowDrip(t *testing.T) {
	dir, err := os.MkdirTemp("", "broker-idle-drip-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	s := NewServer("127.0.0.1:0", dir, 10*1024*1024, 4096)
	s.SetConnReadIdle(2 * time.Second)
	s.SetConnMaxLifetime(60 * time.Second)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	time.Sleep(150 * time.Millisecond)
	addr := s.Addr()

	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := conn.Write([]byte{0x00, 0x00, 0x00, 0x10}); err != nil {
			return
		}
		for range 12 {
			if _, err := conn.Write([]byte{0x01}); err != nil {
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if s.connCount.Load() == 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	<-done

	if s.connCount.Load() != 0 {
		t.Fatalf("conn count %d, want 0 after read idle on partial frame drip", s.connCount.Load())
	}
}

func TestPartialFrame_ReadIdleClosesAfterSingleByte(t *testing.T) {
	dir, err := os.MkdirTemp("", "broker-idle-byte-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	s := NewServer("127.0.0.1:0", dir, 10*1024*1024, 4096)
	s.SetConnReadIdle(1 * time.Second)
	s.SetConnMaxLifetime(60 * time.Second)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	time.Sleep(150 * time.Millisecond)

	conn, err := net.DialTimeout("tcp", s.Addr(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	if _, err := conn.Write([]byte{0x00, 0x00, 0x00, 0x10}); err != nil {
		t.Fatal(err)
	}

	time.Sleep(1500 * time.Millisecond)

	buf := make([]byte, 1)
	_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, err = conn.Read(buf)
	if err == nil {
		t.Fatal("expected read error after server idle close")
	}
}

func TestPartialFrame_ReadIdle_IncrementsMetric(t *testing.T) {
	dir, err := os.MkdirTemp("", "broker-idle-metric-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	before := testutil.ToFloat64(metrics.BrokerConnIdleCloseTotal.WithLabelValues("read_idle"))

	s := NewServer("127.0.0.1:0", dir, 10*1024*1024, 4096)
	s.SetConnReadIdle(2 * time.Second)
	s.SetConnMaxLifetime(60 * time.Second)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	time.Sleep(150 * time.Millisecond)

	conn, err := net.DialTimeout("tcp", s.Addr(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := conn.Write([]byte{0x00, 0x00, 0x00, 0x10}); err != nil {
			return
		}
		for range 12 {
			if _, err := conn.Write([]byte{0x01}); err != nil {
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()

	waitUntil := time.Now().Add(5 * time.Second)
	for time.Now().Before(waitUntil) {
		if s.connCount.Load() == 0 &&
			testutil.ToFloat64(metrics.BrokerConnIdleCloseTotal.WithLabelValues("read_idle")) > before {
			<-done
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	<-done
	after := testutil.ToFloat64(metrics.BrokerConnIdleCloseTotal.WithLabelValues("read_idle"))
	if after <= before {
		t.Fatalf("read_idle metric before=%v after=%v conn=%d", before, after, s.connCount.Load())
	}
}

func TestMaxConnLifetime_ClosesPartialFrame(t *testing.T) {
	dir, err := os.MkdirTemp("", "broker-max-life-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	before := testutil.ToFloat64(metrics.BrokerConnIdleCloseTotal.WithLabelValues("max_lifetime"))

	s := NewServer("127.0.0.1:0", dir, 10*1024*1024, 4096)
	s.SetConnReadIdle(30 * time.Second)
	s.SetConnMaxLifetime(1 * time.Second)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	time.Sleep(150 * time.Millisecond)

	conn, err := net.DialTimeout("tcp", s.Addr(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for _, b := range []byte{0x00, 0x00, 0x00, 0x10} {
			if _, err := conn.Write([]byte{b}); err != nil {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
		for range 30 {
			if _, err := conn.Write([]byte{0x01}); err != nil {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	waitUntil := time.Now().Add(3 * time.Second)
	for time.Now().Before(waitUntil) {
		if testutil.ToFloat64(metrics.BrokerConnIdleCloseTotal.WithLabelValues("max_lifetime")) > before {
			<-done
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	<-done
	t.Fatalf("max_lifetime metric unchanged before=%v conn=%d", before, s.connCount.Load())
}

func TestPartialFrame_ProgressResetsReadIdle(t *testing.T) {
	dir, err := os.MkdirTemp("", "broker-idle-progress-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	s := NewServer("127.0.0.1:0", dir, 10*1024*1024, 4096)
	s.SetConnReadIdle(2 * time.Second)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	time.Sleep(150 * time.Millisecond)

	cli := client.NewClient(s.Addr(), 2*time.Second)
	if err := cli.Connect(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cli.Close() }()

	if _, err := cli.Produce(context.Background(), "idle-reset-topic", 0, []byte("payload")); err != nil {
		t.Fatal(err)
	}

	time.Sleep(500 * time.Millisecond)
	if s.connCount.Load() == 0 {
		t.Fatal("expected client connection to remain open after full frame")
	}
}
