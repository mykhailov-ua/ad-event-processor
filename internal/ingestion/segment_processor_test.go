package ingestion

import (
	"context"
	"testing"
	"time"

	"espx/internal/domain"
	"espx/pkg/piihash"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestSegmentConversionHandler_skipsNonConversion(t *testing.T) {
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()
	hasher := piihash.TestHasher()
	segmentID := uuid.New()
	repo := &segmentTestRepo{
		camp: &domain.Campaign{
			ID:                uuid.New(),
			RetargetSegmentID: segmentID,
			SegmentTTLHours:   24,
		},
	}
	h := NewSegmentConversionHandler(repo, nil, []redis.UniversalClient{rdb}, hasher)
	h.Handle(&domain.Event{Type: "click", UserID: "u1"}, "1-0")

	ctx := context.Background()
	member, err := segmentMemberExists(ctx, []redis.UniversalClient{rdb}, segmentID, hasher.HashUserID("u1"))
	require.NoError(t, err)
	require.False(t, member)
}

func TestSegmentConversionHandler_addsMember(t *testing.T) {
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()
	hasher := piihash.TestHasher()
	segmentID := uuid.New()
	repo := &segmentTestRepo{
		camp: &domain.Campaign{
			ID:                uuid.New(),
			RetargetSegmentID: segmentID,
			SegmentTTLHours:   24,
		},
	}
	h := NewSegmentConversionHandler(repo, nil, []redis.UniversalClient{rdb}, hasher)
	h.Handle(&domain.Event{
		Type:   conversionEventType,
		UserID: "u1",
	}, "1-0")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	member, err := segmentMemberExists(ctx, []redis.UniversalClient{rdb}, segmentID, hasher.HashUserID("u1"))
	require.NoError(t, err)
	require.True(t, member)
}

func TestSegmentFilter_includeRequiresMembership(t *testing.T) {
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()
	hasher := piihash.TestHasher()
	segmentID := uuid.New()
	userID := "include-user"
	userHash := hasher.HashUserID(userID)
	ctx := context.Background()
	require.NoError(t, addSegmentMember(ctx, []redis.UniversalClient{rdb}, segmentID, userHash, time.Hour))

	campID := uuid.New()
	reg := &segmentTestRegistry{
		camps: map[uuid.UUID]*domain.Campaign{
			campID: {ID: campID, SegmentIncludeID: segmentID},
		},
	}
	filter := NewSegmentFilter([]redis.UniversalClient{rdb}, reg, hasher)
	evt := &domain.Event{CampaignID: campID, UserID: userID}
	require.NoError(t, filter.Check(ctx, evt))
	require.ErrorIs(t, filter.Check(ctx, &domain.Event{CampaignID: campID, UserID: "other"}), ErrSegmentNotIncluded)
}
