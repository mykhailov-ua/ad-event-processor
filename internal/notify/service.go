package notify

import (
	"context"
	"errors"
	"fmt"

	"espx/internal/notify/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool                *pgxpool.Pool
	queries             *db.Queries
	providers           map[db.NotifierProvider]Provider
	options             ServiceOptions
	rateLimiter         *recipientRateLimiter
	deliveryRateLimiter *providerRateLimiter
}

func NewService(pool *pgxpool.Pool, providers map[db.NotifierProvider]Provider) *Service {
	return NewServiceWithOptions(pool, providers, defaultServiceOptions())
}

func NewServiceWithOptions(pool *pgxpool.Pool, providers map[db.NotifierProvider]Provider, opts ServiceOptions) *Service {
	return &Service{
		pool:                pool,
		queries:             db.New(pool),
		providers:           providers,
		options:             opts,
		rateLimiter:         newRecipientRateLimiter(opts.RateLimitPerMinute),
		deliveryRateLimiter: newProviderRateLimiter(deliveryRateLimitsFromOptions(opts)),
	}
}

func (service *Service) GetNotification(ctx context.Context, notificationID string) (Notification, error) {
	id, err := pgUUIDFromString(notificationID)
	if err != nil {
		return Notification{}, err
	}

	row, err := service.queries.GetNotification(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Notification{}, ErrNotificationNotFound
		}
		return Notification{}, fmt.Errorf("query notification: %w", err)
	}

	return notificationFromDB(row), nil
}

func (service *Service) findActiveByDedupKey(ctx context.Context, dedupKey string) (db.NotifierNotification, bool, error) {
	existing, err := service.queries.FindActiveNotificationByDedupKey(ctx, db.FindActiveNotificationByDedupKeyParams{
		DedupKey: pgtype.Text{String: dedupKey, Valid: true},
		Column2:  int64(service.options.dedupCooldown().Seconds()),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.NotifierNotification{}, false, nil
		}
		return db.NotifierNotification{}, false, fmt.Errorf("find active notification by dedup key: %w", err)
	}
	return existing, true, nil
}

func uuidString(id pgtype.UUID) string {
	return uuid.UUID(id.Bytes).String()
}

func pgtypeInt4(value int32) pgtype.Int4 {
	return pgtype.Int4{Int32: value, Valid: true}
}

func pgtypeText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: true}
}

func pgtypeTextOptional(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: value, Valid: true}
}
