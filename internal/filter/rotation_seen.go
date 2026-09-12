package filter

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

const defaultRotationSeenTTL = 30 * 24 * time.Hour

type RotationSelectContext struct {
	CampaignID uuid.UUID
	VisitorKey string
	Redis      redis.UniversalClient
	Now        time.Time
}

func RotationSeenRedisKey(campaignID uuid.UUID, visitorKey string) string {
	if campaignID == uuid.Nil || visitorKey == "" {
		return ""
	}
	return fmt.Sprintf("%s:rot:seen:%s", campaignID.String(), visitorKey)
}

func rotationSeenMember(kind byte, id uuid.UUID) string {
	return string([]byte{kind}) + ":" + id.String()
}

func loadRotationSeen(ctx context.Context, rdb redis.UniversalClient, key string) (map[string]struct{}, error) {
	seen := make(map[string]struct{})
	if rdb == nil || key == "" {
		return seen, nil
	}
	members, err := rdb.SMembers(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	for _, member := range members {
		seen[member] = struct{}{}
	}
	return seen, nil
}

func markRotationSeen(ctx context.Context, rdb redis.UniversalClient, key, member string, ttl time.Duration) error {
	if rdb == nil || key == "" || member == "" {
		return nil
	}
	pipe := rdb.Pipeline()
	pipe.SAdd(ctx, key, member)
	if ttl > 0 {
		pipe.Expire(ctx, key, ttl)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func clearRotationSeen(ctx context.Context, rdb redis.UniversalClient, key string) error {
	if rdb == nil || key == "" {
		return nil
	}
	return rdb.Del(ctx, key).Err()
}

func selectUnseenLander(
	landers []FlowLanderEntry,
	seen map[string]struct{},
	bucket uint32,
	reset bool,
) (int, FlowLanderEntry, bool) {
	if len(landers) == 0 {
		return -1, FlowLanderEntry{}, false
	}
	if reset {
		seen = make(map[string]struct{})
	}
	candidates := make([]FlowLanderEntry, 0, len(landers))
	indexes := make([]int, 0, len(landers))
	for i, lander := range landers {
		if len(lander.URL) == 0 {
			continue
		}
		member := rotationSeenMember('l', lander.LanderID)
		if _, ok := seen[member]; ok {
			continue
		}
		candidates = append(candidates, lander)
		indexes = append(indexes, i)
	}
	if len(candidates) == 0 && !reset {
		return selectUnseenLander(landers, seen, bucket, true)
	}
	if len(candidates) == 0 {
		return -1, FlowLanderEntry{}, false
	}
	idx, picked := selectWeightedLander(candidates, bucket)
	if idx < 0 {
		return -1, FlowLanderEntry{}, false
	}
	return indexes[idx], picked, true
}

func selectUnseenOffer(
	offers []FlowOfferEntry,
	seen map[string]struct{},
	bucket uint32,
	exclude map[uuid.UUID]struct{},
	reset bool,
) (int, FlowOfferEntry, bool) {
	if len(offers) == 0 {
		return -1, FlowOfferEntry{}, false
	}
	if reset {
		seen = make(map[string]struct{})
	}
	candidates := make([]FlowOfferEntry, 0, len(offers))
	indexes := make([]int, 0, len(offers))
	for i, offer := range offers {
		if offer.OfferID != uuid.Nil {
			if exclude != nil {
				if _, skip := exclude[offer.OfferID]; skip {
					continue
				}
			}
			if offer.Capped {
				continue
			}
		}
		if offer.OfferID != uuid.Nil {
			member := rotationSeenMember('o', offer.OfferID)
			if _, ok := seen[member]; ok {
				continue
			}
		}
		candidates = append(candidates, offer)
		indexes = append(indexes, i)
	}
	if len(candidates) == 0 && !reset {
		return selectUnseenOffer(offers, seen, bucket, exclude, true)
	}
	if len(candidates) == 0 {
		return -1, FlowOfferEntry{}, false
	}
	idx, picked := selectWeightedOfferExcluding(candidates, bucket, nil)
	if idx < 0 {
		return -1, FlowOfferEntry{}, false
	}
	return indexes[idx], picked, true
}

func applyRotationSeen(
	ctx context.Context,
	rot *RotationSelectContext,
	landerID, offerID uuid.UUID,
) error {
	if rot == nil || rot.Redis == nil || rot.VisitorKey == "" || rot.CampaignID == uuid.Nil {
		return nil
	}
	key := RotationSeenRedisKey(rot.CampaignID, rot.VisitorKey)
	ttl := defaultRotationSeenTTL
	if landerID != uuid.Nil {
		if err := markRotationSeen(ctx, rot.Redis, key, rotationSeenMember('l', landerID), ttl); err != nil {
			return err
		}
	}
	if offerID != uuid.Nil {
		if err := markRotationSeen(ctx, rot.Redis, key, rotationSeenMember('o', offerID), ttl); err != nil {
			return err
		}
	}
	return nil
}

func selectFixOnLander(landers []FlowLanderEntry, seen map[string]struct{}, bucket uint32) (int, FlowLanderEntry, bool) {
	for i, lander := range landers {
		if len(lander.URL) == 0 {
			continue
		}
		if _, ok := seen[rotationSeenMember('l', lander.LanderID)]; ok {
			return i, lander, true
		}
	}
	idx, lander := selectWeightedLander(landers, bucket)
	return idx, lander, idx >= 0
}

func selectFixOnOffer(offers []FlowOfferEntry, seen map[string]struct{}, exclude map[uuid.UUID]struct{}, bucket uint32) (int, FlowOfferEntry, bool) {
	for i, offer := range offers {
		if offer.OfferID == uuid.Nil {
			continue
		}
		if exclude != nil {
			if _, skip := exclude[offer.OfferID]; skip {
				continue
			}
		}
		if offer.Capped {
			continue
		}
		if _, ok := seen[rotationSeenMember('o', offer.OfferID)]; ok {
			return i, offer, true
		}
	}
	idx, offer := selectWeightedOfferExcluding(offers, bucket, exclude)
	return idx, offer, idx >= 0
}

func selectSequentialLander(landers []FlowLanderEntry) (int, FlowLanderEntry, bool) {
	for i, lander := range landers {
		if lander.LanderID == uuid.Nil || len(lander.URL) == 0 || lander.Weight <= 0 {
			continue
		}
		return i, lander, true
	}
	return -1, FlowLanderEntry{}, false
}

func selectSequentialOffer(offers []FlowOfferEntry, exclude map[uuid.UUID]struct{}) (int, FlowOfferEntry, bool) {
	for i, offer := range offers {
		if offer.OfferID == uuid.Nil || offer.Capped || offer.Weight <= 0 {
			continue
		}
		if offerExcluded(offer.OfferID, exclude) {
			continue
		}
		return i, offer, true
	}
	return -1, FlowOfferEntry{}, false
}

func selectRotationLander(
	path FlowPath,
	seen map[string]struct{},
	userID []byte,
) (int, FlowLanderEntry, bool) {
	bucket := fnv1a32Salted(userID, 'l')
	switch domain.NormalizeRotationMode(path.RotationMode) {
	case domain.RotationModeUnseen:
		return selectUnseenLander(path.Landers, seen, bucket, false)
	case domain.RotationModeFixOn:
		return selectFixOnLander(path.Landers, seen, bucket)
	case domain.RotationModeSequential:
		return selectSequentialLander(path.Landers)
	default:
		idx, lander := selectWeightedLander(path.Landers, bucket)
		return idx, lander, idx >= 0
	}
}

func selectRotationOffer(
	path FlowPath,
	snap *FlowPathSnapshot,
	seen map[string]struct{},
	userID []byte,
	exclude map[uuid.UUID]struct{},
) (int, FlowOfferEntry, bool) {
	mode := domain.NormalizeRotationMode(path.RotationMode)
	bucket := fnv1a32Salted(userID, 'o')
	if snap.RoutingMode == domain.FlowRoutingModeWaterfall ||
		snap.RoutingMode == domain.FlowRoutingModeWaterfallThenLanding {
		idx, offer := selectWaterfallOffer(path.Offers, exclude)
		return idx, offer, idx >= 0
	}
	switch mode {
	case domain.RotationModeUnseen:
		return selectUnseenOffer(path.Offers, seen, bucket, exclude, false)
	case domain.RotationModeFixOn:
		return selectFixOnOffer(path.Offers, seen, exclude, bucket)
	case domain.RotationModeSequential:
		return selectSequentialOffer(path.Offers, exclude)
	default:
		idx, offer := selectWeightedOfferExcluding(path.Offers, bucket, exclude)
		return idx, offer, idx >= 0
	}
}
