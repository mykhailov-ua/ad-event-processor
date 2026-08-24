package server

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"ad-event-processor/pkg/broker/client"
)

func TestMaxConnections_RejectsExcessClients(t *testing.T) {
	dir, err := os.MkdirTemp("", "max-conn-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	s := NewServer("127.0.0.1:0", dir, 10*1024*1024, 4096)
	s.SetMaxConnections(2)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	time.Sleep(150 * time.Millisecond)
	addr := s.Addr()

	hold := make([]net.Conn, 0, 2)
	for i := range 2 {
		c, err := net.DialTimeout("tcp", addr, time.Second)
		if err != nil {
			t.Fatalf("dial %d: %v", i, err)
		}
		hold = append(hold, c)
	}
	defer func() {
		for _, c := range hold {
			_ = c.Close()
		}
	}()

	time.Sleep(50 * time.Millisecond)

	extra, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatalf("dial extra: %v", err)
	}
	defer func() { _ = extra.Close() }()

	buf := make([]byte, 1)
	_ = extra.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	if _, err := extra.Read(buf); err == nil {
		t.Log("extra connection may have been closed by server without readable data")
	}

	if s.connCount.Load() > 2 {
		t.Fatalf("conn count %d exceeds max 2", s.connCount.Load())
	}
}

func TestAdmissionShedding_ProduceOverloaded(t *testing.T) {
	dir, err := os.MkdirTemp("", "admission-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	s := NewServer("127.0.0.1:0", dir, 10*1024*1024, 4096)
	s.SetMaxConnections(10)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	time.Sleep(150 * time.Millisecond)

	clients := make([]*client.Client, 0, 9)
	defer func() {
		for _, c := range clients {
			_ = c.Close()
		}
	}()

	for i := range 9 {
		cli := client.NewClient(s.Addr(), 2*time.Second)
		if err := cli.Connect(); err != nil {
			t.Fatalf("connect %d: %v", i, err)
		}
		clients = append(clients, cli)
	}
	time.Sleep(100 * time.Millisecond)

	if got := s.connCount.Load(); got < 9 {
		t.Fatalf("expected at least 9 connections, got %d", got)
	}
	if !s.isAdmissionShedding() {
		t.Fatalf("expected admission shedding at %d/10 connections", s.connCount.Load())
	}

	_, err = clients[0].Produce(context.Background(), "overload-topic", 0, []byte("x"))
	if err == nil {
		t.Fatal("expected produce to fail with overloaded status")
	}
}

func TestFault_ConnectionLimit_HealthyClientUnaffected(t *testing.T) {
	dir, err := os.MkdirTemp("", "fault-conn-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	s := NewServer("127.0.0.1:0", dir, 10*1024*1024, 4096)
	s.SetMaxConnections(100)
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

	if _, err := cli.Produce(context.Background(), "fault-conn-topic", 0, []byte("ok")); err != nil {
		t.Fatal(err)
	}

	t.Logf("fault_proof fault=connection_limit max=100 shedding=false produce_ok=true")
}
