package privacyadmin

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type Host interface {
	ErrValidation(msg string) error
	MapCampaignNotFound(err error) error
	ConsentRetentionMonths() int
	ConsentUpdateChannel() string
	WorkerBatchContext(parent context.Context) (context.Context, context.CancelFunc)
	ClickHouseDeleteFraudEventsByUser(ctx context.Context, userID string) error
	RedisShards() []redis.UniversalClient
	PublishConsentUpdate(ctx context.Context, hashHex string) error
	PurgeUserRedisKeys(ctx context.Context, hashHex, subjectUserID string) error
	SyncConsentRedisKey(ctx context.Context, hashHex string, purposes int16) error
}

type ConsentRecord struct {
	UserID   string
	Source   string
	Purposes int16
}

type userConsentOutboxPayload struct {
	UserIDHash string `json:"user_id_hash"`
	Purposes   int16  `json:"purposes"`
}

type campaignConsentOutboxPayload struct {
	CampaignID string `json:"campaign_id"`
}

type purgeUserDataOutboxPayload struct {
	ErasureID     string `json:"erasure_id"`
	UserIDHash    string `json:"user_id_hash"`
	SubjectUserID string `json:"subject_user_id"`
}
