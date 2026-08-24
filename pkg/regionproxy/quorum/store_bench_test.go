package quorum

import (
	"context"
	"testing"

	"ad-event-processor/internal/database"

	"github.com/google/uuid"
)

func BenchmarkQuorumBook(b *testing.B) {
	ctx := context.Background()
	redisClient, cleanup := database.SetupTestRedis(b)
	defer cleanup()

	opBase := uuid.New()
	nodes := []string{"proxy-a", "proxy-b", "proxy-c"}

	b.ReportAllocs()
	benchN := 0
	for b.Loop() {
		opID := opBase
		opID[15] = byte(benchN)
		opID[14] = byte(benchN >> 8)
		if _, err := Book(ctx, redisClient, toBenchBytes(opID), nodes, "proxy-a"); err != nil {
			b.Fatal(err)
		}
		benchN++
	}
}

func toBenchBytes(id uuid.UUID) [16]byte {
	var out [16]byte
	copy(out[:], id[:])
	return out
}
