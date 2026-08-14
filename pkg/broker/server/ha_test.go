package server

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/broker/client"

	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestHAClusterFailoverAndReplication(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	redisContainer, err := rediscontainer.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Fatalf("failed to start redis container: %v", err)
	}
	defer func() {
		_ = redisContainer.Terminate(ctx)
	}()

	redisEndpoint, err := redisContainer.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("failed to get redis endpoint: %v", err)
	}
	redisURL := fmt.Sprintf("redis://%s/0", redisEndpoint)

	dir1, err := os.MkdirTemp("", "broker-ha-1-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir1)

	dir2, err := os.MkdirTemp("", "broker-ha-2-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir2)

	s1 := NewServer("127.0.0.1:0", dir1, 10*1024*1024, 4096)
	if err := s1.Start(); err != nil {
		t.Fatal(err)
	}
	defer s1.Stop()

	coord1, err := NewCoordinator("broker-1", s1.Addr(), redisURL, s1)
	if err != nil {
		t.Fatal(err)
	}
	s1.SetCoordinator(coord1)
	coord1.Start(context.Background())
	defer coord1.Stop()

	s2 := NewServer("127.0.0.1:0", dir2, 10*1024*1024, 4096)
	if err := s2.Start(); err != nil {
		t.Fatal(err)
	}
	defer s2.Stop()

	coord2, err := NewCoordinator("broker-2", s2.Addr(), redisURL, s2)
	if err != nil {
		t.Fatal(err)
	}
	s2.SetCoordinator(coord2)
	coord2.Start(context.Background())
	defer coord2.Stop()

	topic := "ha-events"
	pk := topicPartitionKey(topic)

	if _, err := s1.getOrCreatePartition(pk); err != nil {
		t.Fatal(err)
	}
	if _, err := s2.getOrCreatePartition(pk); err != nil {
		t.Fatal(err)
	}

	requireEventually(t, func() bool {
		return coord1.IsLeader(pk) || coord2.IsLeader(pk)
	}, 15*time.Second, 200*time.Millisecond, "expected one broker to be elected as leader")

	l1 := coord1.IsLeader(pk)

	var leaderServer *Server
	var followerServer *Server
	var leaderCoord *Coordinator

	if l1 {
		leaderServer = s1
		followerServer = s2
		leaderCoord = coord1
	} else {
		leaderServer = s2
		followerServer = s1
		leaderCoord = coord2
	}

	cli := client.NewClient(leaderServer.Addr(), 2*time.Second)
	cli.SetRedisURL(redisURL)
	if err := cli.Connect(); err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	msgCount := 20
	for i := range msgCount {
		payload := []byte(fmt.Sprintf("ha-msg-payload-%d", i))
		offset, err := cli.Produce(context.Background(), topic, 0, payload)
		if err != nil {
			t.Fatalf("produce failed on message %d: %v", i, err)
		}
		if offset != uint64(i) {
			t.Errorf("unexpected offset: got %d, expected %d", offset, i)
		}
	}

	requireEventually(t, func() bool {
		fPartition, err := followerServer.getOrCreatePartition(pk)
		if err != nil {
			return false
		}
		return fPartition.NextOffset() == uint64(msgCount)
	}, 10*time.Second, 200*time.Millisecond, "follower must replicate leader messages")

	leaderCoord.Stop()
	leaderServer.Stop()

	requireEventually(t, func() bool {
		if l1 {
			return coord2.IsLeader(pk) && coord2.IsLeaderReady(pk)
		}
		return coord1.IsLeader(pk) && coord1.IsLeaderReady(pk)
	}, 20*time.Second, 200*time.Millisecond, "survivor must become ready leader after failover")

	payload := []byte("msg-after-failover")
	offset, err := cli.Produce(context.Background(), topic, 0, payload)
	if err != nil {
		t.Fatalf("failover produce failed: %v", err)
	}
	expectedOffset := uint64(msgCount)
	if offset != expectedOffset {
		t.Errorf("unexpected offset after failover: got %d, expected %d", offset, expectedOffset)
	}

	newLeaderPartition, err := followerServer.getOrCreatePartition(pk)
	if err != nil {
		t.Fatal(err)
	}
	if newLeaderPartition.NextOffset() != expectedOffset+1 {
		t.Errorf("new leader next offset mismatch: got %d, expected %d", newLeaderPartition.NextOffset(), expectedOffset+1)
	}
}
