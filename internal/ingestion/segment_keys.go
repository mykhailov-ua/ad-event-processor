package ingestion

import (
	"context"
	"errors"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/piihash"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

func appendHex16(dst []byte, h [16]byte) []byte {
	for i := range 16 {
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

func pickSegmentShard(redisShards []redis.UniversalClient, segmentID uuid.UUID) redis.UniversalClient {
	if len(redisShards) == 0 {
		return nil
	}
	var h uint32
	for i := range 16 {
		h = h*31 + uint32(segmentID[i])
	}
	start := int(h % uint32(len(redisShards)))
	for i := range redisShards {
		idx := (start + i) % len(redisShards)
		if redisShards[idx] != nil {
			return redisShards[idx]
		}
	}
	return nil
}

func segmentMemberExists(ctx context.Context, redisShards []redis.UniversalClient, segmentID uuid.UUID, userHash [16]byte) (bool, error) {
	redisClient := pickSegmentShard(redisShards, segmentID)
	if redisClient == nil || segmentID == uuid.Nil {
		return false, nil
	}
	w := bufPool.Get().(*bufWrapper)
	w.buf = appendSegmentMemberKey(w.buf[:0], segmentID, userHash)
	key := unsafeString(w.buf)
	err := redisClient.Get(ctx, key).Err()
	bufPool.Put(w)
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func addSegmentMember(ctx context.Context, redisShards []redis.UniversalClient, segmentID uuid.UUID, userHash [16]byte, ttl time.Duration) error {
	if segmentID == uuid.Nil || ttl <= 0 {
		return nil
	}
	redisClient := pickSegmentShard(redisShards, segmentID)
	if redisClient == nil {
		return nil
	}
	w := bufPool.Get().(*bufWrapper)
	w.buf = appendSegmentMemberKey(w.buf[:0], segmentID, userHash)
	key := unsafeString(w.buf)
	err := redisClient.Set(ctx, key, "1", ttl).Err()
	bufPool.Put(w)
	return err
}

func segmentUserHash(hasher *piihash.Hasher, evt *domain.Event) ([16]byte, bool) {
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
