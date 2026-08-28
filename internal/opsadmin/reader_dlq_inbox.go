package opsadmin

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func DLQInboxSourceFromProvider(provider string) string {
	return dlqInboxSourceFromProvider(provider)
}

func dlqInboxSourceFromProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "facebook", "google", "tiktok", "taboola", "outbrain", "microsoft_ads":
		return "capi"
	default:
		return "postback"
	}
}

func (r *Reader) ListDLQInbox(ctx context.Context, source, cursor string, limit int) (DLQInboxListResult, error) {
	if limit <= 0 {
		limit = 50
	}
	source = strings.ToLower(strings.TrimSpace(source))

	var (
		out        DLQInboxListResult
		merged     []DLQInboxEntryDTO
		nextCursor string
	)

	includeStream := source == "" || source == "stream"
	includePostback := source == "" || source == "postback" || source == "capi"

	if includeStream {
		stream, err := r.listDLQEntries(ctx, cursor, limit)
		if err != nil {
			out.Errors = append(out.Errors, FanOutSourceError{Source: "stream", Code: "SOURCE_UNAVAILABLE"})
		} else {
			out.Errors = append(out.Errors, stream.Errors...)
			if stream.Partial {
				out.Partial = true
			}
			nextCursor = stream.NextCursor
			for _, row := range stream.Items {
				merged = append(merged, DLQInboxEntryDTO{
					ID:         row.ID,
					Source:     "stream",
					CampaignID: row.CampaignID,
					EventType:  row.EventType,
					Error:      row.Error,
					FailedAt:   row.FailedAt,
					Status:     "",
					RetryCount: row.RetryCount,
					ShardID:    row.ShardID,
					StreamID:   row.StreamID,
					EntryID:    row.EntryID,
				})
			}
		}
	}

	if includePostback {
		postbackRows, err := r.listPostbackDLQInbox(ctx, source, limit)
		if err != nil {
			out.Errors = append(out.Errors, FanOutSourceError{Source: "postback", Code: "SOURCE_UNAVAILABLE"})
		} else {
			merged = append(merged, postbackRows...)
		}
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].FailedAt > merged[j].FailedAt
	})
	if len(merged) > limit {
		merged = merged[:limit]
	}
	out.Items = merged
	out.NextCursor = nextCursor
	if out.Items == nil {
		out.Items = []DLQInboxEntryDTO{}
	}
	return out, nil
}

func (r *Reader) listPostbackDLQInbox(ctx context.Context, sourceFilter string, limit int) ([]DLQInboxEntryDTO, error) {
	if r == nil || r.pool() == nil || r.pool() == nil {
		return nil, fmt.Errorf("postgres pool not configured")
	}
	rows, err := r.pool().Query(ctx, `
		SELECT q.id, q.campaign_id, q.click_id, q.event_type, q.failures_count,
		 COALESCE(q.last_error, ''), q.status, q.created_at,
		 COALESCE(pc.provider, 'webhook') AS provider
		FROM postback_dlq q
		LEFT JOIN postback_configs pc ON pc.campaign_id = q.campaign_id
		ORDER BY q.created_at DESC
		LIMIT $1`, limit*2)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []DLQInboxEntryDTO
	for rows.Next() {
		var (
			id         int64
			campaignID uuid.UUID
			clickID    string
			eventType  string
			failures   int32
			lastError  string
			status     string
			createdAt  time.Time
			provider   string
		)
		if err := rows.Scan(&id, &campaignID, &clickID, &eventType, &failures, &lastError, &status, &createdAt, &provider); err != nil {
			return nil, err
		}
		src := dlqInboxSourceFromProvider(provider)
		if sourceFilter == "postback" && src != "postback" {
			continue
		}
		if sourceFilter == "capi" && src != "capi" {
			continue
		}
		items = append(items, DLQInboxEntryDTO{
			ID:         strconv.FormatInt(id, 10),
			Source:     src,
			CampaignID: campaignID.String(),
			ClickID:    clickID,
			EventType:  eventType,
			Error:      lastError,
			FailedAt:   createdAt.UTC().Format(time.RFC3339),
			Status:     status,
			RetryCount: failures,
			Provider:   provider,
		})
		if len(items) >= limit {
			break
		}
	}
	return items, rows.Err()
}

func (r *Reader) RetryDLQInbox(ctx context.Context, source, id, idempotencyKey string) error {
	source = strings.ToLower(strings.TrimSpace(source))
	switch source {
	case "stream":
		shardID := parseInboxStreamShard(id)
		entryID := parseInboxStreamEntryID(id)
		return r.enqueueDLQRetry(ctx, DLQRetryPayload{
			ShardID: shardID,
			EntryID: entryID,
			DLQID:   id,
		}, idempotencyKey)
	case "postback", "capi":
		return r.retryPostbackDLQ(ctx, id)
	default:
		return errInvalidQuery("invalid dlq source")
	}
}

func ParseInboxStreamShard(dlqID string) int {
	return parseInboxStreamShard(dlqID)
}

func parseInboxStreamShard(dlqID string) int {
	const prefix = "shard-"
	if !strings.HasPrefix(dlqID, prefix) {
		return 0
	}
	rest := dlqID[len(prefix):]
	dash := strings.Index(rest, "-")
	if dash <= 0 {
		return 0
	}
	n, err := strconv.Atoi(rest[:dash])
	if err != nil {
		return 0
	}
	return n
}

func ParseInboxStreamEntryID(dlqID string) string {
	return parseInboxStreamEntryID(dlqID)
}

func parseInboxStreamEntryID(dlqID string) string {
	const prefix = "shard-"
	if !strings.HasPrefix(dlqID, prefix) {
		return ""
	}
	rest := dlqID[len(prefix):]
	dash := strings.Index(rest, "-")
	if dash < 0 || dash+1 >= len(rest) {
		return ""
	}
	return rest[dash+1:]
}

func (r *Reader) retryPostbackDLQ(ctx context.Context, id string) error {
	if r == nil || r.pool() == nil || r.pool() == nil {
		return fmt.Errorf("postgres pool not configured")
	}
	dlqID, err := strconv.ParseInt(strings.TrimSpace(id), 10, 64)
	if err != nil {
		return errInvalidQuery("invalid postback dlq id")
	}

	tx, err := r.pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := db.New(tx)
	dlq, err := q.GetPostbackDLQ(ctx, dlqID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrDLQEntryNotFound
		}
		return err
	}
	if dlq.Status == "RETRIED" {
		return errInvalidQuery("already retried")
	}

	_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType: "SEND_POSTBACK",
		Payload:   dlq.Payload,
	})
	if err != nil {
		return err
	}
	err = q.UpdatePostbackDLQ(ctx, db.UpdatePostbackDLQParams{
		ID:            dlq.ID,
		FailuresCount: dlq.FailuresCount,
		LastError:     pgtype.Text{String: "Manual retry triggered", Valid: true},
		Status:        "RETRIED",
	})
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
