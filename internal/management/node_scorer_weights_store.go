package management

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"espx/internal/config"
	db "espx/internal/ingestion/sqlc"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const scoringWeightsReloadInterval = 60 * time.Second

type ScoringWeightsStore struct {
	defs atomic.Pointer[map[string][]ScoringMetricDef]
}

func ValidateScoringWeightsConfig(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) error {
	if cfg == nil || !cfg.MultiRegionEnabled {
		return nil
	}
	_, err := loadScoringWeightsFromSources(ctx, pool, cfg)
	if err != nil {
		return fmt.Errorf("validate scoring weights config: %w", err)
	}
	return nil
}

func NewScoringWeightsStore(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) (*ScoringWeightsStore, error) {
	store := &ScoringWeightsStore{}
	if err := store.reload(ctx, pool, cfg); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *ScoringWeightsStore) Start(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) {
	if s == nil {
		return
	}
	ticker := time.NewTicker(scoringWeightsReloadInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.reload(ctx, pool, cfg); err != nil {
				slog.Warn("scoring weights reload failed", "error", err)
			}
		}
	}
}

func (s *ScoringWeightsStore) MetricsForRole(role string) []ScoringMetricDef {
	if s == nil {
		return MetricsForRole(role)
	}
	m := s.defs.Load()
	if m == nil {
		return MetricsForRole(role)
	}
	if defs, ok := (*m)[role]; ok {
		return defs
	}
	return MetricsForRole(role)
}

func (s *ScoringWeightsStore) reload(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) error {
	byRole, err := loadScoringWeightsFromSources(ctx, pool, cfg)
	if err != nil {
		return err
	}
	defs := BuildScoringMetricDefsByRole(byRole)
	s.defs.Store(&defs)
	return nil
}

func loadScoringWeightsFromSources(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) (ScoringWeightsByRole, error) {
	raw := ""
	if cfg != nil {
		raw = trimJSON(cfg.ScoringWeightsJSON)
	}
	if raw == "" && pool != nil {
		q := db.New(pool)
		val, err := q.GetSystemSetting(ctx, scoringWeightsSettingKey)
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("load scoring weights from system_settings: %w", err)
			}
		} else {
			raw = trimJSON(val)
		}
	}
	if raw == "" {
		return nil, nil
	}
	return ParseScoringWeightsJSON(raw)
}
