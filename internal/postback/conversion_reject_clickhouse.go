package postback

import (
	"context"
	"time"

	"ad-event-processor/internal/database"

	"github.com/google/uuid"
)

const conversionRejectCHTimeout = 15 * time.Second

type clickSnapshot struct {
	clickID    string
	campaignID uuid.UUID
	createdAt  time.Time
	country    string
}

type conversionGoalKey struct {
	campaignID uuid.UUID
	clickID    string
	goalName   string
}

type chConversionClickStore struct {
	ch *database.CHQuery
}

func NewCHConversionClickStore(ch *database.CHQuery) *chConversionClickStore {
	return newCHConversionClickStore(ch)
}

func newCHConversionClickStore(ch *database.CHQuery) *chConversionClickStore {
	if ch == nil {
		return nil
	}
	return &chConversionClickStore{ch: ch}
}

func (s *chConversionClickStore) LoadClicks(ctx context.Context, clickIDs []string) (map[string]clickSnapshot, error) {
	if s == nil || s.ch == nil || len(clickIDs) == 0 {
		return nil, nil
	}
	chCtx, cancel := context.WithTimeout(ctx, conversionRejectCHTimeout)
	defer cancel()

	rows, err := s.ch.Query(chCtx, `
SELECT click_id, campaign_id, created_at, country
FROM clicks
WHERE click_id IN (?)
`, clickIDs)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]clickSnapshot, len(clickIDs))
	for rows.Next() {
		var snap clickSnapshot
		var country string
		if err := rows.Scan(&snap.clickID, &snap.campaignID, &snap.createdAt, &country); err != nil {
			return nil, err
		}
		snap.country = normalizeCountryCode(country)
		out[snap.clickID] = snap
	}
	return out, rows.Err()
}

func (s *chConversionClickStore) LoadExistingGoals(ctx context.Context, keys []conversionGoalKey) (map[conversionGoalKey]struct{}, error) {
	if s == nil || s.ch == nil || len(keys) == 0 {
		return nil, nil
	}
	clickIDs := make([]string, 0, len(keys))
	campaignIDs := make([]uuid.UUID, 0, len(keys))
	seenClick := make(map[string]struct{}, len(keys))
	seenCamp := make(map[uuid.UUID]struct{}, len(keys))
	for _, key := range keys {
		if key.clickID == "" || key.campaignID == uuid.Nil {
			continue
		}
		if _, ok := seenClick[key.clickID]; !ok {
			seenClick[key.clickID] = struct{}{}
			clickIDs = append(clickIDs, key.clickID)
		}
		if _, ok := seenCamp[key.campaignID]; !ok {
			seenCamp[key.campaignID] = struct{}{}
			campaignIDs = append(campaignIDs, key.campaignID)
		}
	}
	if len(clickIDs) == 0 {
		return nil, nil
	}

	chCtx, cancel := context.WithTimeout(ctx, conversionRejectCHTimeout)
	defer cancel()

	rows, err := s.ch.Query(chCtx, `
SELECT click_id, campaign_id, JSONExtractString(payload, 'goal_name') AS goal_name
FROM conversions
WHERE click_id IN (?)
 AND campaign_id IN (?)
`, clickIDs, campaignIDs)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make(map[conversionGoalKey]struct{})
	for rows.Next() {
		var key conversionGoalKey
		var goal string
		if err := rows.Scan(&key.clickID, &key.campaignID, &goal); err != nil {
			return nil, err
		}
		key.goalName = normalizeGoalName(goal)
		out[key] = struct{}{}
	}
	return out, rows.Err()
}
