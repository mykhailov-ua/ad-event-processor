package fraudadmin

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
	"github.com/jackc/pgx/v5/pgxpool"
)

const policyPresetCacheTTL = 5 * time.Minute

type policyPresetRow struct {
	name      string
	pass      uint8
	suspect   uint8
	ivt       uint8
	block     uint8
	updatedAt time.Time
}

var policyPresetCache struct {
	mu       sync.RWMutex
	loadedAt time.Time
	rows     []policyPresetRow
}

func InvalidatePolicyPresetCache() {
	policyPresetCache.mu.Lock()
	policyPresetCache.loadedAt = time.Time{}
	policyPresetCache.rows = nil
	policyPresetCache.mu.Unlock()
}

func policyPresetDTO(row policyPresetRow) FraudPolicyPresetDTO {
	return FraudPolicyPresetDTO{
		Name:      row.name,
		Pass:      row.pass,
		Suspect:   row.suspect,
		IVT:       row.ivt,
		Block:     row.block,
		UpdatedAt: row.updatedAt.UTC().Format(time.RFC3339),
	}
}

func loadPolicyPresetsFromPG(ctx context.Context, pool *pgxpool.Pool) ([]policyPresetRow, error) {
	if pool == nil {
		return nil, fmt.Errorf("postgres pool not configured")
	}
	rows, err := pool.Query(ctx, `
		SELECT name, pass, suspect, ivt, block, updated_at
		FROM fraud_policy_presets
		ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("query fraud_policy_presets: %w", err)
	}
	defer rows.Close()

	out := make([]policyPresetRow, 0, 8)
	for rows.Next() {
		var row policyPresetRow
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

func cachedPolicyPresetRows(ctx context.Context, pool *pgxpool.Pool) ([]policyPresetRow, error) {
	now := time.Now()
	policyPresetCache.mu.RLock()
	if len(policyPresetCache.rows) > 0 && now.Sub(policyPresetCache.loadedAt) < policyPresetCacheTTL {
		out := append([]policyPresetRow(nil), policyPresetCache.rows...)
		policyPresetCache.mu.RUnlock()
		return out, nil
	}
	policyPresetCache.mu.RUnlock()

	rows, err := loadPolicyPresetsFromPG(ctx, pool)
	if err != nil {
		return nil, err
	}

	policyPresetCache.mu.Lock()
	policyPresetCache.rows = append([]policyPresetRow(nil), rows...)
	policyPresetCache.loadedAt = now
	policyPresetCache.mu.Unlock()
	return rows, nil
}

func ListPolicyPresets(ctx context.Context, pool *pgxpool.Pool) ([]FraudPolicyPresetDTO, error) {
	rows, err := cachedPolicyPresetRows(ctx, pool)
	if err != nil {
		return DefaultPolicyPresetDTOs(), nil
	}
	out := make([]FraudPolicyPresetDTO, len(rows))
	for i, row := range rows {
		out[i] = policyPresetDTO(row)
	}
	return out, nil
}

func DefaultPolicyPresetDTOs() []FraudPolicyPresetDTO {
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

func ResolvePresetThresholds(ctx context.Context, pool *pgxpool.Pool, name string) (uint8, uint8, uint8, uint8, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return 0, 0, 0, 0, ValidationError("preset must be conservative, balanced, aggressive, enhanced_defense, or social_in_app")
	}
	rows, err := cachedPolicyPresetRows(ctx, pool)
	if err == nil {
		for _, row := range rows {
			if row.name == name {
				return row.pass, row.suspect, row.ivt, row.block, nil
			}
		}
	}
	pass, suspect, ivt, block, ok := domain.ResolveFraudPreset(name)
	if !ok {
		return 0, 0, 0, 0, ValidationError("preset must be conservative, balanced, aggressive, enhanced_defense, or social_in_app")
	}
	return pass, suspect, ivt, block, nil
}

type PresetsHost interface {
	PresetsPool() *pgxpool.Pool
	PresetAuditUpdate(ctx context.Context, q db.Querier, adminID uuid.UUID, name string, pass, suspect, ivt, block uint8)
	PresetActorID(ctx context.Context) uuid.UUID
}

func UpdatePolicyPreset(ctx context.Context, host PresetsHost, name string, req PatchFraudPolicyPresetRequest) (FraudPolicyPresetDTO, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return FraudPolicyPresetDTO{}, ValidationError("preset name is required")
	}
	if host == nil || host.PresetsPool() == nil {
		return FraudPolicyPresetDTO{}, fmt.Errorf("postgres pool not configured")
	}

	var out FraudPolicyPresetDTO
	err := pgx.BeginFunc(ctx, host.PresetsPool(), func(tx pgx.Tx) error {
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
				return ValidationError("unknown fraud policy preset")
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
		if err := ValidateThresholds(nextPass, nextSuspect, nextIVT, nextBlock); err != nil {
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

		host.PresetAuditUpdate(ctx, q, host.PresetActorID(ctx), name, nextPass, nextSuspect, nextIVT, nextBlock)

		out = policyPresetDTO(policyPresetRow{
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
	InvalidatePolicyPresetCache()
	return out, nil
}
