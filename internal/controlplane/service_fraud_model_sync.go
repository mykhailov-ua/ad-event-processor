package controlplane

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

func (s *Service) CheckAndHandleStaleEpochs(ctx context.Context) error {
	if len(s.rdbs) == 0 {
		return nil
	}

	now := time.Now().Unix()
	var maxAppliedAt int64
	var foundStale bool

	for _, rdb := range s.rdbs {
		if rdb == nil {
			continue
		}
		val, err := rdb.Get(ctx, "ml:model:applied_at").Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			return fmt.Errorf("query ml:model:applied_at: %w", err)
		}
		appliedAt, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			continue
		}
		if appliedAt > maxAppliedAt {
			maxAppliedAt = appliedAt
		}
	}

	if maxAppliedAt > 0 && now-maxAppliedAt > 600 {
		foundStale = true
	}

	if !foundStale {
		return nil
	}

	var currentPctStr string
	err := s.pool.QueryRow(ctx, "SELECT value FROM system_settings WHERE key = 'fraud_rl_suspect_pct'").Scan(&currentPctStr)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			currentPctStr = "50"
		} else {
			return err
		}
	}

	currentPct, _ := strconv.Atoi(currentPctStr)
	newPct := currentPct / 2
	if newPct < 10 {
		newPct = 10
	}

	if newPct == currentPct {
		return nil
	}

	if err := s.UpdateSettings(ctx, map[string]string{
		"fraud_rl_suspect_pct": strconv.Itoa(newPct),
	}); err != nil {
		return fmt.Errorf("tighten suspect rate limit: %w", err)
	}
	return nil
}
