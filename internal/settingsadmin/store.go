package settingsadmin

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
	host Host
}

func NewStore(pool *pgxpool.Pool, host Host) *Store {
	return &Store{pool: pool, host: host}
}

func (st *Store) poolOrNil() *pgxpool.Pool {
	if st == nil {
		return nil
	}
	return st.pool
}

func (st *Store) BlockIP(ctx context.Context, ip string, source string) error {
	return st.BlockIPWithTTL(ctx, ip, source, nil)
}

func (st *Store) PreviewBlockIP(ctx context.Context, ip string, source string, ttlSeconds *int64) (MutationPreview, error) {
	return st.blockIPWithTTL(ctx, ip, source, ttlSeconds, true)
}

func (st *Store) BlockIPWithTTL(ctx context.Context, ip string, source string, ttlSeconds *int64) error {
	_, err := st.blockIPWithTTL(ctx, ip, source, ttlSeconds, false)
	return err
}

func (st *Store) blockIPWithTTL(ctx context.Context, ip string, source string, ttlSeconds *int64, dryRun bool) (MutationPreview, error) {
	if st.host.IsProtectedIP(ip) {
		return MutationPreview{}, fmt.Errorf("IP %s is protected by allowlist", ip)
	}

	reason := normalizeBlacklistReason(source)
	expiresAt := resolveBlacklistExpiry(reason, ttlSeconds, blacklistTTLFromHost(st.host))

	if dryRun {
		change := BlockIPPreviewChange{
			IP:          ip,
			Reason:      reason,
			OutboxEvent: "UPDATE_BLACKLIST",
			Action:      "add",
		}
		if expiresAt.Valid {
			change.ExpiresAt = expiresAt.Time.UTC().Format(time.RFC3339)
		}
		return st.host.NewBlockIPPreview(change)
	}

	err := pgx.BeginFunc(ctx, st.poolOrNil(), func(tx pgx.Tx) error {
		q := db.New(tx)
		_, err := q.CreateBlacklistIP(ctx, db.CreateBlacklistIPParams{
			Ip:        ip,
			Reason:    reason,
			ExpiresAt: expiresAt,
		})
		if err != nil {
			return err
		}

		ttlVal := pgtypeInt4FromExpiry(expiresAt)

		_, err = q.CreateEdgeBlockAudit(ctx, db.CreateEdgeBlockAuditParams{
			Ip:       ip,
			ReasonID: reason,
			Ttl:      ttlVal,
			Source:   source,
		})
		if err != nil {
			return err
		}

		st.host.AuditLog(ctx, q, st.host.ActorUserID(ctx), "BLOCK_IP", "system", nil, map[string]string{"ip": ip, "source": reason}, nil)

		payload, err := coldpath.MarshalOutbox(blacklistOutboxPayload{Action: "add", IP: ip, Reason: reason})
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

func pgtypeInt4FromExpiry(expiresAt pgtype.Timestamptz) pgtype.Int4 {
	if !expiresAt.Valid {
		return pgtype.Int4{}
	}
	diff := expiresAt.Time.Sub(time.Now().UTC())
	if diff <= 0 {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(diff.Seconds()), Valid: true}
}

func (st *Store) EnqueueFraudThreat(ctx context.Context, action, ip, campaignID string, score float64, boost int32, ttlSeconds int64) error {
	_, err := st.EnqueueFraudThreatBatch(ctx, []FraudThreatItem{{
		Action:     action,
		IP:         ip,
		CampaignID: campaignID,
		Score:      score,
		Boost:      boost,
		TTLSeconds: ttlSeconds,
	}})
	return err
}

func fraudThreatOutboxEventType(action string) (string, error) {
	switch action {
	case "boost":
		return "ML_SCORE_BOOST", nil
	case "silent_reject", "ghost":
		return "ML_SILENT_REJECT", nil
	case "blacklist":
		return "ML_BLACKLIST_ADD", nil
	default:
		return "", fmt.Errorf("unknown ml threat action: %s", action)
	}
}

func (st *Store) EnqueueFraudThreatBatch(ctx context.Context, items []FraudThreatItem) (int, error) {
	if len(items) == 0 {
		return 0, st.host.ErrValidation("items required")
	}
	if len(items) > fraudadmin.ThreatBatchMax {
		return 0, st.host.ErrValidation(fmt.Sprintf("max %d items per bulk request", fraudadmin.ThreatBatchMax))
	}

	var inserted int
	err := pgx.BeginFunc(ctx, st.poolOrNil(), func(tx pgx.Tx) error {
		q := db.New(tx)
		eventTypes := make([]string, 0, len(items))
		payloads := make([][]byte, 0, len(items))

		for i, item := range items {
			if item.Action == "" || item.CampaignID == "" {
				return st.host.ErrValidation(fmt.Sprintf("row %d: action and campaign_id required", i+1))
			}
			if _, err := uuid.Parse(item.CampaignID); err != nil {
				return st.host.ErrValidation(fmt.Sprintf("row %d: invalid campaign_id format", i+1))
			}

			eventType, err := fraudThreatOutboxEventType(item.Action)
			if err != nil {
				return err
			}

			payload, err := coldpath.MarshalOutbox(fraudThreatOutboxPayload(item))
			if err != nil {
				return fmt.Errorf("row %d: marshal ml threat payload: %w", i+1, err)
			}

			eventTypes = append(eventTypes, eventType)
			payloads = append(payloads, payload)
		}

		if err := q.CreateOutboxEventsBatch(ctx, db.CreateOutboxEventsBatchParams{
			EventTypes: eventTypes,
			Payloads:   payloads,
		}); err != nil {
			return err
		}
		inserted = len(items)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return inserted, nil
}

func (st *Store) UnblockExpiredBlacklist(ctx context.Context, rows []db.ListExpiredBlacklistIPsRow) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	ips := make([]string, len(rows))
	for i, row := range rows {
		ips[i] = row.Ip
	}
	var removed int
	err := pgx.BeginFunc(ctx, st.poolOrNil(), func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `DELETE FROM ip_blacklist WHERE ip = ANY($1)`, ips)
		if err != nil {
			return err
		}
		removed = int(tag.RowsAffected())

		q := db.New(tx)
		st.host.AuditLog(ctx, q, st.host.ActorUserID(ctx), "BLACKLIST_JANITOR", "system", nil, map[string]any{
			"count": removed,
			"ips":   ips,
		}, nil)

		batch := &pgx.Batch{}
		for _, row := range rows {
			reason := normalizeBlacklistReason(row.Reason)
			payload, err := coldpath.MarshalOutbox(blacklistOutboxPayload{Action: "remove", IP: row.Ip, Reason: reason})
			if err != nil {
				return fmt.Errorf("marshal blacklist outbox payload: %w", err)
			}
			batch.Queue(
				`INSERT INTO outbox_events (event_type, payload) VALUES ($1, $2)`,
				"UPDATE_BLACKLIST",
				payload,
			)
		}
		br := tx.SendBatch(ctx, batch)
		for range rows {
			if _, err := br.Exec(); err != nil {
				_ = br.Close()
				return err
			}
		}
		return br.Close()
	})
	return removed, err
}

func (st *Store) UnblockIP(ctx context.Context, ip string, source string) error {
	reason := normalizeBlacklistReason(source)

	return pgx.BeginFunc(ctx, st.poolOrNil(), func(tx pgx.Tx) error {
		q := db.New(tx)
		err := q.DeleteBlacklistIP(ctx, ip)
		if err != nil {
			return err
		}

		st.host.AuditLog(ctx, q, st.host.ActorUserID(ctx), "UNBLOCK_IP", "system", nil, map[string]string{"ip": ip, "source": reason}, nil)

		payload, err := coldpath.MarshalOutbox(blacklistOutboxPayload{Action: "remove", IP: ip, Reason: reason})
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

func (st *Store) UpdateSettings(ctx context.Context, settings map[string]string) error {
	normalized, err := normalizeSystemSettings(settings)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, st.poolOrNil(), func(tx pgx.Tx) error {
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

		st.host.AuditLog(ctx, q, st.host.ActorUserID(ctx), "UPDATE_SETTINGS", "system", nil, normalized, nil)
		payloadBytes, err := coldpath.MarshalOutbox(settingsOutboxPayload{Settings: normalized})
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
			norm, err := domain.NormalizeRtbBudgetAuthoritySetting(v)
			if err != nil {
				return nil, err
			}
			out[k] = norm
			continue
		}
		if k == domain.SystemSettingRtbMode {
			norm, err := domain.NormalizeRtbModeSetting(v)
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

func (st *Store) ListBlacklist(ctx context.Context, limit, offset int32) ([]BlacklistEntry, int64, error) {
	q := db.New(st.poolOrNil())
	listParams := db.ListBlacklistParams{Limit: limit, Offset: offset}
	return coldpath.PaginatedList(
		func() (int64, error) { return q.CountBlacklist(ctx) },
		func() ([]db.IpBlacklist, error) { return q.ListBlacklist(ctx, listParams) },
		blacklistToDTO,
	)
}

func blacklistToDTO(r db.IpBlacklist) BlacklistEntry {
	createdAt := r.CreatedAt.Time.Format(time.RFC3339)
	dto := BlacklistEntry{
		ID:               r.ID,
		IP:               r.Ip,
		Reason:           r.Reason,
		CreatedAt:        createdAt,
		CreatedAtDisplay: coldpath.RFC3339Display(createdAt),
	}
	if r.ExpiresAt.Valid {
		expiresAt := r.ExpiresAt.Time.UTC().Format(time.RFC3339)
		dto.ExpiresAt = expiresAt
		dto.ExpiresAtDisplay = coldpath.RFC3339Display(expiresAt)
	}
	return dto
}

func (st *Store) GetSettings(ctx context.Context) (map[string]string, error) {
	q := db.New(st.poolOrNil())
	rows, err := q.GetAllSystemSettings(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, r := range rows {
		out[r.Key] = r.Value
	}
	return out, nil
}

func (st *Store) SyncSystemState(ctx context.Context) error {
	q := db.New(st.poolOrNil())

	bl, err := q.GetAllBlacklist(ctx)
	if err != nil {
		return fmt.Errorf("failed to get blacklist from db: %w", err)
	}

	if len(st.host.RedisShards()) == 0 {
		return fmt.Errorf("no redis client available")
	}

	reasonIPs := make(map[string][]any)
	for _, item := range bl {
		reason := normalizeBlacklistReason(item.Reason)
		reasonIPs[reason] = append(reasonIPs[reason], item.Ip)
	}

	for reason, ips := range reasonIPs {
		key := "blacklist:" + reason
		if err := st.host.SyncGlobalSetReplace(ctx, key, ips); err != nil {
			return fmt.Errorf("failed to sync blacklist key %s: %w", key, err)
		}
	}

	settingsRows, err := q.GetAllSystemSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to get settings from db: %w", err)
	}

	if len(settingsRows) > 0 {
		settingsMap := make(map[string]string)
		for _, r := range settingsRows {
			settingsMap[r.Key] = r.Value
		}
		if err := st.host.SyncGlobalConfig(ctx, settingsMap); err != nil {
			return fmt.Errorf("failed to sync settings to redis: %w", err)
		}
		if err := st.host.ReplicateConfigVersionFromPrimary(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (st *Store) ToggleEmergencyBreaker(ctx context.Context, active bool, reason string) error {
	val := "false"
	if active {
		val = "true"
	}

	return pgx.BeginFunc(ctx, st.poolOrNil(), func(tx pgx.Tx) error {
		q := db.New(tx)
		err := q.SetSystemSetting(ctx, db.SetSystemSettingParams{
			Key:   "emergency_breaker",
			Value: val,
		})
		if err != nil {
			return err
		}

		st.host.AuditEmergencyBreaker(ctx, q, st.host.ActorUserID(ctx), active, reason)

		settings := map[string]string{
			"emergency_breaker": val,
		}
		payloadBytes, err := coldpath.MarshalOutbox(settingsOutboxPayload{Settings: settings})
		if err != nil {
			return fmt.Errorf("marshal emergency breaker outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "UPDATE_SETTINGS",
			Payload:   payloadBytes,
		})
		return err
	})
}
