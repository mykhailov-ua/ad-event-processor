package quorum

import (
	"context"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/database"

	"github.com/google/uuid"
)

func BenchmarkQuorumBook(b *testing.B) {
	ctx := context.Background()
	rdb, cleanup := database.SetupTestRedis(b)
	defer cleanup()

	opBase := uuid.New()
	nodes := []string{"proxy-a", "proxy-b", "proxy-c"}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		opID := opBase
		opID[15] = byte(i)
		opID[14] = byte(i >> 8)
		if _, err := Book(ctx, rdb, toBenchBytes(opID), nodes, "proxy-a"); err != nil {
			b.Fatal(err)
		}
	}
}

func toBenchBytes(id uuid.UUID) [16]byte {
	var out [16]byte
	copy(out[:], id[:])
	return out
}
