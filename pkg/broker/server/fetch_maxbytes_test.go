package server

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/pkg/broker/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetch_MaxBytesBoundary(t *testing.T) {
	srv := NewServer("127.0.0.1:0", t.TempDir(), 1024*1024, 4096)
	require.NoError(t, srv.Start())
	defer srv.Stop()

	cli := client.NewClient(srv.Addr(), 2*time.Second)
	require.NoError(t, cli.Connect())
	defer func() { _ = cli.Close() }()

	small := make([]byte, 32)
	large := make([]byte, 256)
	for i := range small {
		small[i] = 's'
	}
	for i := range large {
		large[i] = 'l'
	}

	_, err := cli.Produce(context.Background(), "tracker-logs", 0, small)
	require.NoError(t, err)
	_, err = cli.Produce(context.Background(), "tracker-logs", 0, large)
	require.NoError(t, err)

	iter, err := cli.Fetch(context.Background(), "tracker-logs", 0, 0, 60)
	require.NoError(t, err)

	count := 0
	for iter.Next() {
		count++
		if count == 1 {
			assert.Equal(t, small, iter.Payload)
		}
	}
	assert.Equal(t, 1, count, "maxBytes must stop before oversized second record")

	iter2, err := cli.Fetch(context.Background(), "tracker-logs", 0, 1, 512)
	require.NoError(t, err)
	require.True(t, iter2.Next())
	assert.Equal(t, large, iter2.Payload)
	assert.False(t, iter2.Next())
}
