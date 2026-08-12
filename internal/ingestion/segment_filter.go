package ingestion

import (
	"context"
	"errors"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/pkg/piihash"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

var (
	ErrSegmentExcluded    = errors.New("segment excluded")
	ErrSegmentNotIncluded = errors.New("segment not included")
)

type SegmentFilter struct {
	rdbs     []redis.UniversalClient
	registry domain.CampaignRegistry
	hasher   *piihash.Hasher
}

func NewSegmentFilter(rdbs []redis.UniversalClient, registry domain.CampaignRegistry, hasher *piihash.Hasher) *SegmentFilter {
	return &SegmentFilter{
		rdbs:     rdbs,
		registry: registry,
		hasher:   hasher,
	}
}

func (f *SegmentFilter) Check(ctx context.Context, evt *domain.Event) error {
	if evt == nil || f.registry == nil {
		return nil
	}
	camp, ok := f.registry.GetCampaign(evt.CampaignID)
	if !ok || camp == nil {
		return nil
	}
	if camp.SegmentIncludeID == uuid.Nil && camp.SegmentExcludeID == uuid.Nil {
		return nil
	}
	userHash, ok := segmentUserHash(f.hasher, evt)
	if !ok {
		if camp.SegmentIncludeID != uuid.Nil {
			return ErrSegmentNotIncluded
		}
		return nil
	}
	if camp.SegmentExcludeID != uuid.Nil {
		member, err := segmentMemberExists(ctx, f.rdbs, camp.SegmentExcludeID, userHash)
		if err != nil {
			return nil
		}
		if member {
			return ErrSegmentExcluded
		}
	}
	if camp.SegmentIncludeID != uuid.Nil {
		member, err := segmentMemberExists(ctx, f.rdbs, camp.SegmentIncludeID, userHash)
		if err != nil {
			return nil
		}
		if !member {
			return ErrSegmentNotIncluded
		}
	}
	return nil
}
