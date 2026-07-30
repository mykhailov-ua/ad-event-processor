package management

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"espx/internal/edge/allowlist"
	"espx/internal/ingestion"
	db "espx/internal/ingestion/sqlc"
	"espx/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type BlacklistDTO struct {
	ID        int64  `json:"id"`
	IP        string `json:"ip"`
	Reason    string `json:"reason"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

func (s *Service) BlockIP(ctx context.Context, ip string, source string) error {
	return s.BlockIPWithTTL(ctx, ip, source, nil)
}

func (s *Service) PreviewBlockIP(ctx context.Context, ip string, source string, ttlSeconds *int64) (MutationPreview, error) {
	return s.blockIPWithTTL(ctx, ip, source, ttlSeconds, true)
}

func (s *Service) BlockIPWithTTL(ctx context.Context, ip string, source string, ttlSeconds *int64) error {
	_, err := s.blockIPWithTTL(ctx, ip, source, ttlSeconds, false)
	return err
}

func (s *Service) blockIPWithTTL(ctx context.Context, ip string, source string, ttlSeconds *int64, dryRun bool) (MutationPreview, error) {
	if allowlist.IsProtected(ip) {
		return MutationPreview{}, fmt.Errorf("IP %s is protected by allowlist", ip)
	}

	reason := normalizeBlacklistReason(source)
	expiresAt := resolveBlacklistExpiry(reason, ttlSeconds, blacklistTTLFromConfig(s.cfg))

	if dryRun {
		preview := MutationPreview{
			DryRun: true,
			Action: "BLOCK_IP",
			WouldChange: map[string]any{
				"ip":           ip,
				"reason":       reason,
				"outbox_event": "UPDATE_BLACKLIST",
				"action":       "add",
			},
		}
		if expiresAt.Valid {
			preview.WouldChange["expires_at"] = expiresAt.Time.UTC().Format(time.RFC3339)
		}
		return preview, nil
	}

	err := pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		_, err := q.CreateBlacklistIP(ctx, db.CreateBlacklistIPParams{
			Ip:        ip,
			Reason:    reason,
			ExpiresAt: expiresAt,
		})
		if err != nil {
			return err
		}

		var ttlVal pgtype.Int4
		if expiresAt.Valid {
			diff := expiresAt.Time.Sub(time.Now().UTC())
			if diff > 0 {
				ttlVal = pgtype.Int4{Int32: int32(diff.Seconds()), Valid: true}
			}
		}

		_, err = q.CreateEdgeBlockAudit(ctx, db.CreateEdgeBlockAuditParams{
			Ip:       ip,
			ReasonID: reason,
			Ttl:      ttlVal,
			Source:   source,
		})
		if err != nil {
			return err
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "BLOCK_IP", "system", nil, map[string]string{"ip": ip, "source": reason}, nil)

		payload, err := coldpath.MarshalJSON(BlacklistPayload{Action: "add", IP: ip, Reason: reason})
		if err != nil {
			return fmt.Errorf("marshal blacklist outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "UPDATE_BLACKLIST",
			Payload:   payload,
		})
		return err
	})
	return MutationPreview{}, err
}

func (s *Service) EnqueueFraudThreat(ctx context.Context, p FraudThreatPayload) error {
	if _, err := uuid.Parse(p.CampaignID); err != nil {
		return fmt.Errorf("invalid campaign id: %w", err)
	}

	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		payload, err := coldpath.MarshalJSON(p)
		if err != nil {
			return fmt.Errorf("marshal ml threat payload: %w", err)
		}

		var eventType string
		switch p.Action {
		case "boost":
			eventType = "ML_SCORE_BOOST"
		case "ghost":
			eventType = "ML_GHOST_IVT"
		case "blacklist":
			eventType = "ML_BLACKLIST_ADD"
		default:
			return fmt.Errorf("unknown ml threat action: %s", p.Action)
		}

		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: eventType,
			Payload:   payload,
		})
		return err
	})
}

func (s *Service) UnblockIP(ctx context.Context, ip string, source string) error {
	reason := normalizeBlacklistReason(source)

	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		err := q.DeleteBlacklistIP(ctx, ip)
		if err != nil {
			return err
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "UNBLOCK_IP", "system", nil, map[string]string{"ip": ip, "source": reason}, nil)

		payload, err := coldpath.MarshalJSON(BlacklistPayload{Action: "remove", IP: ip, Reason: reason})
		if err != nil {
			return fmt.Errorf("marshal blacklist outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "UPDATE_BLACKLIST",
			Payload:   payload,
		})
		return err
	})
}

func (s *Service) UpdateSettings(ctx context.Context, settings map[string]string) error {
	normalized, err := normalizeSystemSettings(settings)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		for k, v := range normalized {
			err := q.SetSystemSetting(ctx, db.SetSystemSettingParams{
				Key:   k,
				Value: v,
			})
			if err != nil {
				return err
			}
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "UPDATE_SETTINGS", "system", nil, normalized, nil)
		payloadBytes, err := coldpath.MarshalJSON(SettingsPayload{Settings: normalized})
		if err != nil {
			return fmt.Errorf("marshal settings outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "UPDATE_SETTINGS", Payload: payloadBytes})
		return err
	})
}

func normalizeSystemSettings(settings map[string]string) (map[string]string, error) {
	if len(settings) == 0 {
		return settings, nil
	}
	out := make(map[string]string, len(settings))
	for k, v := range settings {
		if k == "rtb_budget_authority" {
			norm, err := ingestion.NormalizeRtbBudgetAuthoritySetting(v)
			if err != nil {
				return nil, err
			}
			out[k] = norm
			continue
		}
		if k == ingestion.SystemSettingRtbMode {
			norm, err := ingestion.NormalizeRtbModeSetting(v)
			if err != nil {
				return nil, err
			}
			out[k] = norm
			continue
		}
		out[k] = v
	}
	return out, nil
}

func (s *Service) ListBlacklist(ctx context.Context, limit, offset int32) ([]BlacklistDTO, int64, error) {
	q := db.New(s.GetPool())
	listParams := db.ListBlacklistParams{Limit: limit, Offset: offset}
	return coldpath.PaginatedList(
		func() (int64, error) { return q.CountBlacklist(ctx) },
		func() ([]db.IpBlacklist, error) { return q.ListBlacklist(ctx, listParams) },
		blacklistToDTO,
	)
}

func blacklistToDTO(r db.IpBlacklist) BlacklistDTO {
	dto := BlacklistDTO{
		ID:        r.ID,
		IP:        r.Ip,
		Reason:    r.Reason,
		CreatedAt: r.CreatedAt.Time.Format(time.RFC3339),
	}
	if r.ExpiresAt.Valid {
		dto.ExpiresAt = r.ExpiresAt.Time.UTC().Format(time.RFC3339)
	}
	return dto
}

func (s *Service) GetSettings(ctx context.Context) (map[string]string, error) {
	q := db.New(s.GetPool())
	rows, err := q.GetAllSystemSettings(ctx)
	if err != nil {
		return nil, err
	}
	return coldpath.KeyByValue(rows, func(r db.GetAllSystemSettingsRow) string { return r.Key }, func(r db.GetAllSystemSettingsRow) string { return r.Value }), nil
}

func (s *Service) SyncSystemState(ctx context.Context) error {
	q := db.New(s.GetPool())

	bl, err := q.GetAllBlacklist(ctx)
	if err != nil {
		return fmt.Errorf("failed to get blacklist from db: %w", err)
	}

	if len(s.rdbs) == 0 {
		return fmt.Errorf("no redis client available")
	}

	reasonIPs := make(map[string][]any)
	for _, item := range bl {
		reason := normalizeBlacklistReason(item.Reason)
		reasonIPs[reason] = append(reasonIPs[reason], item.Ip)
	}

	for reason, ips := range reasonIPs {
		key := "blacklist:" + reason
		for _, rdb := range s.rdbs {
			if err := rdb.Del(ctx, key).Err(); err != nil {
				return fmt.Errorf("failed to reset blacklist key %s: %w", key, err)
			}
			if len(ips) > 0 {
				if err := rdb.SAdd(ctx, key, ips...).Err(); err != nil {
					return fmt.Errorf("failed to sync blacklist key %s: %w", key, err)
				}
			}
		}
	}

	st, err := q.GetAllSystemSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to get settings from db: %w", err)
	}

	if len(st) > 0 {
		settingsMap := make(map[string]string)
		for _, r := range st {
			settingsMap[r.Key] = r.Value
		}
		if err := syncGlobalConfigToAllShards(ctx, s.rdbs, settingsMap, 0); err != nil {
			return fmt.Errorf("failed to sync settings to redis: %w", err)
		}
		if err := replicateConfigVersionFromPrimary(ctx, s.rdbs); err != nil {
			return err
		}
	}

	slog.Info("system state synchronized with redis successfully", "blacklist_items", len(bl), "settings_items", len(st))
	return nil
}

func (s *Service) RunSystemStateSyncer(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	_ = s.SyncSystemState(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.SyncSystemState(ctx); err != nil {
				slog.Error("failed to sync system state", "error", err)
			}
		}
	}
}

func (s *Service) ToggleEmergencyBreaker(ctx context.Context, active bool, reason string) error {
	val := "false"
	if active {
		val = "true"
	}

	err := pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		err := q.SetSystemSetting(ctx, db.SetSystemSettingParams{
			Key:   "emergency_breaker",
			Value: val,
		})
		if err != nil {
			return err
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}

		s.AuditLog(ctx, q, uid, "EMERGENCY_BREAKER_TOGGLED", "system", nil, map[string]any{
			"active": active,
			"reason": reason,
		}, nil)

		settings := map[string]string{
			"emergency_breaker": val,
		}
		payloadBytes, err := coldpath.MarshalJSON(SettingsPayload{Settings: settings})
		if err != nil {
			return fmt.Errorf("marshal emergency breaker outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "UPDATE_SETTINGS",
			Payload:   payloadBytes,
		})
		return err
	})
	return err
}
