package server

import (
	"net"
	"testing"
	"time"

	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/broker/protocol"
	"ad-event-processor/pkg/iogate"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestSlowClient_ReadIdleClosesPartialFrameDrip(t *testing.T) {
	srv, err := NewServer("127.0.0.1:0", t.TempDir(), iogate.NewDiskWriteGate(iogate.TestGateConfig()))
	if err != nil {
		t.Fatal(err)
	}
	srv.SetConnReadIdle(2 * time.Second)
	srv.SetConnMaxLifetime(60 * time.Second)
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	time.Sleep(150 * time.Millisecond)

	conn, err := net.DialTimeout("tcp", srv.Addr(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

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
		if srv.connCount.Load() == 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	<-done

	if srv.connCount.Load() != 0 {
		t.Fatalf("conn count %d, want 0 after read idle on partial frame drip", srv.connCount.Load())
	}
}

func TestReadIdle_ClosesAfterPartialHeader(t *testing.T) {
	srv, err := NewServer("127.0.0.1:0", t.TempDir(), iogate.NewDiskWriteGate(iogate.TestGateConfig()))
	if err != nil {
		t.Fatal(err)
	}
	srv.SetConnReadIdle(1 * time.Second)
	srv.SetConnMaxLifetime(60 * time.Second)
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	time.Sleep(150 * time.Millisecond)

	conn, err := net.DialTimeout("tcp", srv.Addr(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

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

func TestReadIdle_IncrementsMetric(t *testing.T) {
	srv, err := NewServer("127.0.0.1:0", t.TempDir(), iogate.NewDiskWriteGate(iogate.TestGateConfig()))
	if err != nil {
		t.Fatal(err)
	}
	before := testutil.ToFloat64(metrics.RegionProxyConnIdleCloseTotal.WithLabelValues("read_idle"))

	srv.SetConnReadIdle(2 * time.Second)
	srv.SetConnMaxLifetime(60 * time.Second)
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	time.Sleep(150 * time.Millisecond)

	conn, err := net.DialTimeout("tcp", srv.Addr(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

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
		if srv.connCount.Load() == 0 &&
			testutil.ToFloat64(metrics.RegionProxyConnIdleCloseTotal.WithLabelValues("read_idle")) > before {
			<-done
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	<-done
	after := testutil.ToFloat64(metrics.RegionProxyConnIdleCloseTotal.WithLabelValues("read_idle"))
	if after <= before {
		t.Fatalf("read_idle metric before=%v after=%v conn=%d", before, after, srv.connCount.Load())
	}
}

func TestSlowClient_MaxLifetimeClosesPartialFrame(t *testing.T) {
	srv, err := NewServer("127.0.0.1:0", t.TempDir(), iogate.NewDiskWriteGate(iogate.TestGateConfig()))
	if err != nil {
		t.Fatal(err)
	}
	before := testutil.ToFloat64(metrics.RegionProxyConnIdleCloseTotal.WithLabelValues("max_lifetime"))

	srv.SetConnReadIdle(30 * time.Second)
	srv.SetConnMaxLifetime(1 * time.Second)
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	time.Sleep(150 * time.Millisecond)

	conn, err := net.DialTimeout("tcp", srv.Addr(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

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
		if testutil.ToFloat64(metrics.RegionProxyConnIdleCloseTotal.WithLabelValues("max_lifetime")) > before {
			<-done
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	<-done
	t.Fatalf("max_lifetime metric unchanged before=%v conn=%d", before, srv.connCount.Load())
}

func TestReadIdle_ProgressResetsOnFullFrame(t *testing.T) {
	srv, err := NewServer("127.0.0.1:0", t.TempDir(), iogate.NewDiskWriteGate(iogate.TestGateConfig()))
	if err != nil {
		t.Fatal(err)
	}
	srv.SetConnReadIdle(2 * time.Second)
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	time.Sleep(150 * time.Millisecond)

	conn, err := net.DialTimeout("tcp", srv.Addr(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	writeBuf := make([]byte, 4096)
	readBuf := make([]byte, 4096)
	lenBuf := make([]byte, 4)

	reg := protocol.EncodeRegisterTopicRequest(writeBuf, 1, DefaultIngressTopic)
	if _, err := conn.Write(reg); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := protocol.ReadFrame(conn, readBuf, lenBuf); err != nil {
		t.Fatal(err)
	}

	time.Sleep(500 * time.Millisecond)
	if srv.connCount.Load() == 0 {
		t.Fatal("expected client connection to remain open after full frame")
	}
}
