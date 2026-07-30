package ingestion

import (
	"context"
	"time"

	"espx/internal/campaignmodel"
	"espx/pkg/piihash"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	redis "github.com/redis/go-redis/v9"
)

func applyCampaignSegmentFields(
	camp *campaignmodel.Campaign,
	retarget, include, exclude pgtype.UUID,
	ttlHours int32,
) {
	if camp == nil {
		return
	}
	if retarget.Valid {
		camp.RetargetSegmentID = uuid.UUID(retarget.Bytes)
	}
	camp.SegmentTTLHours = ttlHours
	if include.Valid {
		camp.SegmentIncludeID = uuid.UUID(include.Bytes)
	}
	if exclude.Valid {
		camp.SegmentExcludeID = uuid.UUID(exclude.Bytes)
	}
}

func appendHex16(dst []byte, h [16]byte) []byte {
	for i := 0; i < 16; i++ {
		dst = append(dst, hexChars[h[i]>>4], hexChars[h[i]&0xf])
	}
	return dst
}

func appendSegmentMemberKey(dst []byte, segmentID uuid.UUID, userHash [16]byte) []byte {
	dst = append(dst, "segment:u:"...)
	dst = appendUUID(dst, segmentID)
	dst = append(dst, ':')
	return appendHex16(dst, userHash)
}

func pickSegmentShard(rdbs []redis.UniversalClient, segmentID uuid.UUID) redis.UniversalClient {
	if len(rdbs) == 0 {
		return nil
	}
	if len(rdbs) == 1 {
		return rdbs[0]
	}
	var h uint32
	for i := 0; i < 16; i++ {
		h = h*31 + uint32(segmentID[i])
	}
	return rdbs[int(h%uint32(len(rdbs)))]
}

func segmentMemberExists(ctx context.Context, rdbs []redis.UniversalClient, segmentID uuid.UUID, userHash [16]byte) (bool, error) {
	rdb := pickSegmentShard(rdbs, segmentID)
	if rdb == nil || segmentID == uuid.Nil {
		return false, nil
	}
	w := bufPool.Get().(*bufWrapper)
	w.buf = appendSegmentMemberKey(w.buf[:0], segmentID, userHash)
	key := unsafeString(w.buf)
	err := rdb.Get(ctx, key).Err()
	bufPool.Put(w)
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func addSegmentMember(ctx context.Context, rdbs []redis.UniversalClient, segmentID uuid.UUID, userHash [16]byte, ttl time.Duration) error {
	if segmentID == uuid.Nil || ttl <= 0 {
		return nil
	}
	rdb := pickSegmentShard(rdbs, segmentID)
	if rdb == nil {
		return nil
	}
	w := bufPool.Get().(*bufWrapper)
	w.buf = appendSegmentMemberKey(w.buf[:0], segmentID, userHash)
	key := unsafeString(w.buf)
	err := rdb.Set(ctx, key, "1", ttl).Err()
	bufPool.Put(w)
	return err
}

func segmentUserHash(hasher *piihash.Hasher, evt *campaignmodel.Event) ([16]byte, bool) {
	if evt == nil || evt.UserID == "" {
		return [16]byte{}, false
	}
	if evt.HasUserPIIHash {
		return evt.UserPIIHash, true
	}
	if hasher == nil {
		return [16]byte{}, false
	}
	h := hasher.HashUserID(evt.UserID)
	evt.UserPIIHash = h
	evt.HasUserPIIHash = true
	return h, true
}
