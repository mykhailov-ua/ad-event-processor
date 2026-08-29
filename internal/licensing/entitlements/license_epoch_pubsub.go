package entitlements

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	redis "github.com/redis/go-redis/v9"
)

const LicenseEpochPubSubChannel = "license:epoch"

type licenseEpochNotice struct {
	Seq    uint64 `json:"seq"`
	Reason string `json:"reason,omitempty"`
}

var (
	licenseEpochSeq        atomic.Uint64
	licenseEpochSeenSeq    atomic.Uint64
	licenseEpochPublisher  func(context.Context, licenseEpochNotice) error
	licenseEpochSyncWG     sync.WaitGroup
	licenseEpochSyncActive atomic.Bool
)

func StartLicenseEpochSync(ctx context.Context, redisClient redis.UniversalClient) {
	if redisClient == nil || licenseEpochSyncActive.Swap(true) {
		return
	}
	licenseEpochPublisher = func(ctx context.Context, notice licenseEpochNotice) error {
		payload, err := json.Marshal(notice)
		if err != nil {
			return err
		}
		return redisClient.Publish(ctx, LicenseEpochPubSubChannel, payload).Err()
	}
	licenseEpochSyncWG.Add(1)
	go func() {
		defer licenseEpochSyncWG.Done()
		runLicenseEpochSubscriber(ctx, redisClient)
	}()
}

func runLicenseEpochSubscriber(ctx context.Context, redisClient redis.UniversalClient) {
	pubsub := redisClient.Subscribe(ctx, LicenseEpochPubSubChannel)
	defer func() { _ = pubsub.Close() }()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if msg == nil {
				continue
			}
			applyLicenseEpochNotice(msg.Payload)
		}
	}
}

func PublishLicenseEpochNotice(reason string) {
	pub := licenseEpochPublisher
	if pub == nil {
		return
	}
	seq := licenseEpochSeq.Add(1)
	notice := licenseEpochNotice{Seq: seq, Reason: reason}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := pub(ctx, notice); err != nil {
		slog.Debug("license epoch pubsub publish", "error", err, "reason", reason)
	}
}

func applyLicenseEpochNotice(payload string) {
	var notice licenseEpochNotice
	if err := json.Unmarshal([]byte(payload), &notice); err != nil || notice.Seq == 0 {
		invalidateLicenseEpochFromRemote()
		return
	}
	for {
		seen := licenseEpochSeenSeq.Load()
		if notice.Seq <= seen {
			return
		}
		if licenseEpochSeenSeq.CompareAndSwap(seen, notice.Seq) {
			break
		}
	}
	invalidateLicenseEpochFromRemote()
}

func invalidateLicenseEpochFromRemote() {
	licenseEpochInvalid.Store(1)
	PublishFeatureSeed(0, false)
}

func WaitLicenseEpochSyncForTest() {
	licenseEpochSyncWG.Wait()
}

func ResetLicenseEpochPubSubForTest() {
	licenseEpochSyncActive.Store(false)
	licenseEpochPublisher = nil
	licenseEpochSeq.Store(0)
	licenseEpochSeenSeq.Store(0)
}
