package notify

import (
	"context"
	"time"

	"espx/internal/notify/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

func sendTestNotification(ctx context.Context, svc *Service, input NotificationInput) (SendNotificationResult, error) {
	return svc.SendNotificationInput(ctx, input)
}

func getTestNotification(ctx context.Context, svc *Service, notificationID string) (Notification, error) {
	return svc.GetNotification(ctx, notificationID)
}

func newTestConfig() Config {
	return Config{RequireCredentials: false}
}

func newTestBreakers() Breakers {
	return NewBreakers(3, 2, 10*time.Second)
}

func newTestService(pool *pgxpool.Pool) *Service {
	return NewService(pool, newTestConfig(), newTestBreakers())
}

func newTestServiceWithBreakers(pool *pgxpool.Pool, breakers Breakers) *Service {
	return NewService(pool, newTestConfig(), breakers)
}

func newBroadcastTestConfig(failProvider db.NotifierProvider) (Config, Breakers) {
	cfg := newTestConfig()
	switch failProvider {
	case db.NotifierProviderSLACK:
		cfg.FailSlack = true
	case db.NotifierProviderTELEGRAM:
		cfg.FailTelegram = true
	case db.NotifierProviderSMS:
		cfg.FailSMS = true
	}
	return cfg, NewBreakers(10, 2, 10*time.Second)
}

func countNotificationsByStatus(ctx context.Context, pool *pgxpool.Pool, status db.NotifierNotificationStatus) (int, error) {
	var count int
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM notify.notifications WHERE status = $1`, status).Scan(&count)
	return count, err
}
