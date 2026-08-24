package controlplane

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const fraudPolicyPresetCacheTTL = 5 * time.Minute

type FraudPolicyPresetDTO struct {
	Name      string `json:"name"`
	Pass      uint8  `json:"pass"`
	Suspect   uint8  `json:"suspect"`
	IVT       uint8  `json:"ivt"`
	Block     uint8  `json:"block"`
	UpdatedAt string `json:"updated_at"`
}

type PatchFraudPolicyPresetRequest struct {
	Pass    *uint8 `json:"pass,omitempty"`
	Suspect *uint8 `json:"suspect,omitempty"`
	IVT     *uint8 `json:"ivt,omitempty"`
	Block   *uint8 `json:"block,omitempty"`
}

type fraudPolicyPresetRow struct {
	name      string
	pass      uint8
	suspect   uint8
	ivt       uint8
	block     uint8
	updatedAt time.Time
}

var fraudPolicyPresetCache struct {
	mu       sync.RWMutex
	loadedAt time.Time
	rows     []fraudPolicyPresetRow
}

func invalidateFraudPolicyPresetCache() {
	fraudPolicyPresetCache.mu.Lock()
	fraudPolicyPresetCache.loadedAt = time.Time{}
	fraudPolicyPresetCache.rows = nil
	fraudPolicyPresetCache.mu.Unlock()
}

func fraudPolicyPresetDTO(row fraudPolicyPresetRow) FraudPolicyPresetDTO {
	return FraudPolicyPresetDTO{
		Name:      row.name,
		Pass:      row.pass,
		Suspect:   row.suspect,
		IVT:       row.ivt,
		Block:     row.block,
		UpdatedAt: row.updatedAt.UTC().Format(time.RFC3339),
	}
}

func (s *Service) loadFraudPolicyPresetsFromPG(ctx context.Context) ([]fraudPolicyPresetRow, error) {
	if s == nil || s.GetPool() == nil {
		return nil, fmt.Errorf("postgres pool not configured")
	}
	rows, err := s.GetPool().Query(ctx, `
		SELECT name, pass, suspect, ivt, block, updated_at
		FROM fraud_policy_presets
		ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("query fraud_policy_presets: %w", err)
	}
	defer rows.Close()

	out := make([]fraudPolicyPresetRow, 0, 8)
	for rows.Next() {
		var row fraudPolicyPresetRow
		var pass, suspect, ivt, block int16
		if err := rows.Scan(&row.name, &pass, &suspect, &ivt, &block, &row.updatedAt); err != nil {
			return nil, err
		}
		row.pass = uint8(pass)
		row.suspect = uint8(suspect)
		row.ivt = uint8(ivt)
		row.block = uint8(block)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Service) cachedFraudPolicyPresetRows(ctx context.Context) ([]fraudPolicyPresetRow, error) {
	now := time.Now()
	fraudPolicyPresetCache.mu.RLock()
	if len(fraudPolicyPresetCache.rows) > 0 && now.Sub(fraudPolicyPresetCache.loadedAt) < fraudPolicyPresetCacheTTL {
		out := append([]fraudPolicyPresetRow(nil), fraudPolicyPresetCache.rows...)
		fraudPolicyPresetCache.mu.RUnlock()
		return out, nil
	}
	fraudPolicyPresetCache.mu.RUnlock()

	rows, err := s.loadFraudPolicyPresetsFromPG(ctx)
	if err != nil {
		return nil, err
	}

	fraudPolicyPresetCache.mu.Lock()
	fraudPolicyPresetCache.rows = append([]fraudPolicyPresetRow(nil), rows...)
	fraudPolicyPresetCache.loadedAt = now
	fraudPolicyPresetCache.mu.Unlock()
	return rows, nil
}

func (s *Service) ListFraudPolicyPresets(ctx context.Context) ([]FraudPolicyPresetDTO, error) {
	rows, err := s.cachedFraudPolicyPresetRows(ctx)
	if err != nil {
		return defaultFraudPolicyPresetDTOs(), nil
	}
	out := make([]FraudPolicyPresetDTO, len(rows))
	for i, row := range rows {
		out[i] = fraudPolicyPresetDTO(row)
	}
	return out, nil
}

func defaultFraudPolicyPresetDTOs() []FraudPolicyPresetDTO {
	names := []string{
		domain.FraudPresetConservative,
		domain.FraudPresetBalanced,
		domain.FraudPresetAggressive,
		domain.FraudPresetEnhancedDefenseLegacy,
		domain.FraudPresetSocialInApp,
	}
	out := make([]FraudPolicyPresetDTO, 0, len(names))
	for _, name := range names {
		pass, suspect, ivt, block, ok := domain.ResolveFraudPreset(name)
		if !ok {
			continue
		}
		out = append(out, FraudPolicyPresetDTO{
			Name:    name,
			Pass:    pass,
			Suspect: suspect,
			IVT:     ivt,
			Block:   block,
		})
	}
	return out
}

func (s *Service) resolveFraudPresetThresholds(ctx context.Context, name string) (uint8, uint8, uint8, uint8, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return 0, 0, 0, 0, errValidation("preset must be conservative, balanced, aggressive, enhanced_defense, or social_in_app")
	}
	rows, err := s.cachedFraudPolicyPresetRows(ctx)
	if err == nil {
		for _, row := range rows {
			if row.name == name {
				return row.pass, row.suspect, row.ivt, row.block, nil
			}
		}
	}
	pass, suspect, ivt, block, ok := domain.ResolveFraudPreset(name)
	if !ok {
		return 0, 0, 0, 0, errValidation("preset must be conservative, balanced, aggressive, enhanced_defense, or social_in_app")
	}
	return pass, suspect, ivt, block, nil
}

func (s *Service) UpdateFraudPolicyPreset(ctx context.Context, name string, req PatchFraudPolicyPresetRequest) (FraudPolicyPresetDTO, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return FraudPolicyPresetDTO{}, errValidation("preset name is required")
	}
	if s == nil || s.GetPool() == nil {
		return FraudPolicyPresetDTO{}, fmt.Errorf("postgres pool not configured")
	}

	var out FraudPolicyPresetDTO
	err := pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		var pass, suspect, ivt, block int16
		var updatedAt time.Time
		err := tx.QueryRow(ctx, `
			SELECT pass, suspect, ivt, block, updated_at
			FROM fraud_policy_presets
			WHERE name = $1
			FOR UPDATE`, name).Scan(&pass, &suspect, &ivt, &block, &updatedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errValidation("unknown fraud policy preset")
			}
			return err
		}

		nextPass := uint8(pass)
		nextSuspect := uint8(suspect)
		nextIVT := uint8(ivt)
		nextBlock := uint8(block)
		if req.Pass != nil {
			nextPass = *req.Pass
		}
		if req.Suspect != nil {
			nextSuspect = *req.Suspect
		}
		if req.IVT != nil {
			nextIVT = *req.IVT
		}
		if req.Block != nil {
			nextBlock = *req.Block
		}
		if err := validateFraudThresholds(nextPass, nextSuspect, nextIVT, nextBlock); err != nil {
			return err
		}

		err = tx.QueryRow(ctx, `
			UPDATE fraud_policy_presets
			SET pass = $2, suspect = $3, ivt = $4, block = $5, updated_at = NOW()
			WHERE name = $1
			RETURNING pass, suspect, ivt, block, updated_at`,
			name, int16(nextPass), int16(nextSuspect), int16(nextIVT), int16(nextBlock),
		).Scan(&pass, &suspect, &ivt, &block, &updatedAt)
		if err != nil {
			return err
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "UPDATE_FRAUD_POLICY_PRESET", "system", nil, map[string]any{
			"name":    name,
			"pass":    nextPass,
			"suspect": nextSuspect,
			"ivt":     nextIVT,
			"block":   nextBlock,
		}, nil)

		out = fraudPolicyPresetDTO(fraudPolicyPresetRow{
			name:      name,
			pass:      uint8(pass),
			suspect:   uint8(suspect),
			ivt:       uint8(ivt),
			block:     uint8(block),
			updatedAt: updatedAt,
		})
		return nil
	})
	if err != nil {
		return FraudPolicyPresetDTO{}, err
	}
	invalidateFraudPolicyPresetCache()
	return out, nil
}
