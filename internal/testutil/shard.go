package testutil

import (
	"testing"

	"ad-event-processor/internal/ingestion"

	"github.com/google/uuid"
)

func CampaignIDForShard(t testing.TB, sharder ingestion.Sharder, wantShard int) uuid.UUID {
	t.Helper()
	for range 20_000 {
		id := uuid.New()
		if sharder.GetShard(id) == wantShard {
			return id
		}
	}
	t.Fatalf("could not find campaign id for shard %d", wantShard)
	return uuid.Nil
}
