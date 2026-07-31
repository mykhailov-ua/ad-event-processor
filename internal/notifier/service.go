package notifier

import (
	"context"
	"errors"
	"fmt"

	"espx/internal/notifier/db"
	"espx/internal/notifier/pb"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool                *pgxpool.Pool
	queries             *db.Queries
	providers           map[pb.Provider]Provider
	options             ServiceOptions
	rateLimiter         *recipientRateLimiter
	deliveryRateLimiter *providerRateLimiter
}

func NewService(pool *pgxpool.Pool, providers map[pb.Provider]Provider) *Service {
	return NewServiceWithOptions(pool, providers, defaultServiceOptions())
}

func NewServiceWithOptions(pool *pgxpool.Pool, providers map[pb.Provider]Provider, opts ServiceOptions) *Service {
	return &Service{
		pool:                pool,
		queries:             db.New(pool),
		providers:           providers,
		options:             opts,
		rateLimiter:         newRecipientRateLimiter(opts.RateLimitPerMinute),
		deliveryRateLimiter: newProviderRateLimiter(deliveryRateLimitsFromOptions(opts)),
	}
}

func (service *Service) SendNotification(ctx context.Context, req *pb.SendNotificationRequest) (*pb.SendNotificationResponse, error) {
	result, err := service.SendNotificationInput(ctx, NotificationInputFromPB(req))
	if err != nil {
		return nil, err
	}
	return &pb.SendNotificationResponse{
		NotificationId: result.NotificationID,
		Status:         MapDBStatusToPB(result.Status),
		Deduplicated:   result.Deduplicated,
	}, nil
}

func (service *Service) SendNotificationBatch(ctx context.Context, req *pb.SendNotificationBatchRequest) (*pb.SendNotificationBatchResponse, error) {
	if req == nil || len(req.Notifications) == 0 {
		return nil, ErrBatchEmpty
	}

	out := make([]*pb.SendNotificationResponse, 0, len(req.Notifications))
	for _, item := range req.Notifications {
		resp, err := service.SendNotification(ctx, item)
		if err != nil {
			return nil, fmt.Errorf("batch item failed: %w", err)
		}
		out = append(out, resp)
	}
	return &pb.SendNotificationBatchResponse{Notifications: out}, nil
}

func (service *Service) GetNotification(ctx context.Context, req *pb.GetNotificationRequest) (*pb.GetNotificationResponse, error) {
	id, err := pgUUIDFromString(req.NotificationId)
	if err != nil {
		return nil, err
	}

	notification, err := service.queries.GetNotification(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotificationNotFound
		}
		return nil, fmt.Errorf("query notification: %w", err)
	}

	return &pb.GetNotificationResponse{
		Notification: notificationToProto(notification),
	}, nil
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
