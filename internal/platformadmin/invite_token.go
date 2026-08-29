package platformadmin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	teamInviteKeyPrefix = "auth:team_invite:"
	teamInviteTTL       = 72 * time.Hour
)

type TeamInvitePayload struct {
	UserID     uuid.UUID `json:"user_id"`
	CustomerID uuid.UUID `json:"customer_id"`
	Email      string    `json:"email"`
}

func teamInviteRedisKey(token string) string {
	return teamInviteKeyPrefix + token
}

func StoreTeamInvite(ctx context.Context, rdb redis.UniversalClient, token string, payload TeamInvitePayload) error {
	if rdb == nil {
		return errors.New("invite redis unavailable")
	}
	if token == "" {
		return ErrInviteInvalid
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return rdb.Set(ctx, teamInviteRedisKey(token), raw, teamInviteTTL).Err()
}

func LoadTeamInvite(ctx context.Context, rdb redis.UniversalClient, token string) (TeamInvitePayload, error) {
	if rdb == nil {
		return TeamInvitePayload{}, errors.New("invite redis unavailable")
	}
	if token == "" {
		return TeamInvitePayload{}, ErrInviteInvalid
	}
	raw, err := rdb.Get(ctx, teamInviteRedisKey(token)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return TeamInvitePayload{}, ErrInviteInvalid
		}
		return TeamInvitePayload{}, err
	}
	var payload TeamInvitePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return TeamInvitePayload{}, fmt.Errorf("%w: corrupt invite payload", ErrInviteInvalid)
	}
	if payload.UserID == uuid.Nil || payload.CustomerID == uuid.Nil || payload.Email == "" {
		return TeamInvitePayload{}, ErrInviteInvalid
	}
	return payload, nil
}

func DeleteTeamInvite(ctx context.Context, rdb redis.UniversalClient, token string) error {
	if rdb == nil || token == "" {
		return nil
	}
	return rdb.Del(ctx, teamInviteRedisKey(token)).Err()
}
