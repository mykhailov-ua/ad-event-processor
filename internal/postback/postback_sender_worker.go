package postback

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	db "github.com/bidshard/ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/time/rate"
)

var (
	ErrDuplicateEvent          = errors.New("duplicate postback event ignored")
	ErrDispatchFinalizePending = errors.New("dispatch delivered awaiting finalize")
)

type PostbackWorker struct {
	pool               *pgxpool.Pool
	client             *http.Client
	encryptionKey      []byte
	limiters           map[string]*rate.Limiter
	limitersMu         sync.RWMutex
	staleProcessingSec int32
	batchSize          int32

	adapters map[string]PostbackAdapter

	onDispatchAttempt func()
}

func NewPostbackWorker(pool *pgxpool.Pool, encryptionKey []byte) *PostbackWorker {
	switch {
	case len(encryptionKey) == 0:
		encryptionKey = []byte("postback-encryption-secret-key32")
	case len(encryptionKey) < 32:
		padded := make([]byte, 32)
		copy(padded, encryptionKey)
		encryptionKey = padded
	case len(encryptionKey) > 32:
		encryptionKey = encryptionKey[:32]
	}

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	return &PostbackWorker{
		pool:               pool,
		client:             client,
		encryptionKey:      encryptionKey,
		limiters:           make(map[string]*rate.Limiter),
		staleProcessingSec: 120,
		batchSize:          50,
		adapters: map[string]PostbackAdapter{
			"facebook": &FacebookAdapter{},
			"google":   &GoogleAdapter{},
			"tiktok":   &TikTokAdapter{},
			"webhook":  &WebhookAdapter{},
		},
	}
}

func (w *PostbackWorker) ConfigureStaleProcessingSec(sec int32) {
	if w == nil || sec <= 0 {
		return
	}
	w.staleProcessingSec = sec
}

func (w *PostbackWorker) ConfigureBatchSize(size int32) {
	if w == nil || size <= 0 {
		return
	}
	w.batchSize = size
}

func (w *PostbackWorker) Start(ctx context.Context, interval time.Duration) {
	slog.Info("Postback sender worker starting", "interval", interval, "batch_size", w.batchSize)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.ProcessBatch(ctx); err != nil {
				slog.Error("Postback processing batch failed", "error", err)
			}
		}
	}
}

func (w *PostbackWorker) ProcessBatch(ctx context.Context) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := db.New(tx)
	events, err := q.GetPendingPostbackEventsForUpdate(ctx, db.GetPendingPostbackEventsForUpdateParams{
		Limit:   w.batchSize,
		Column2: w.staleProcessingSec,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}

	if len(events) == 0 {
		return nil
	}

	eventIDs := make([]int64, len(events))
	for i, ev := range events {
		eventIDs[i] = ev.ID
	}

	_, err = tx.Exec(ctx, `
		UPDATE outbox_events
		SET status = 'PROCESSING', processing_started_at = NOW()
		WHERE id = ANY($1)`, eventIDs)
	if err != nil {
		return err
	}

	claimed, err := claimPostbackDispatchesInTx(ctx, q, events)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	payloads := make([]PostbackPayload, len(claimed))
	for i, c := range claimed {
		if c.skip {
			continue
		}
		payload, parseErr := parsePostbackPayload(c.event.Payload)
		if parseErr != nil {
			continue
		}
		payloads[i] = payload
	}
	configs, err := w.loadPostbackConfigs(ctx, uniqueCampaignIDsFromEvents(events, payloads))
	if err != nil {
		slog.Warn("postback batch config preload failed, falling back to per-event lookup", "error", err)
	}

	var processedIDs, failedIDs, processingIDs []int64
	for _, c := range claimed {
		if c.skip {
			processedIDs = append(processedIDs, c.event.ID)
			continue
		}
		err := w.ProcessEvent(ctx, c.event, configs)
		if err != nil {
			if errors.Is(err, ErrDispatchFinalizePending) {
				processingIDs = append(processingIDs, c.event.ID)
				slog.Warn("Postback finalize pending, will retry", "id", c.event.ID, "error", err)
				continue
			}
			slog.Warn("Failed to process postback event", "id", c.event.ID, "error", err)
			failedIDs = append(failedIDs, c.event.ID)
		} else {
			processedIDs = append(processedIDs, c.event.ID)
		}
	}
	w.batchUpdateOutboxStatus(ctx, processedIDs, failedIDs, processingIDs)

	return nil
}

func (w *PostbackWorker) ProcessEvent(ctx context.Context, ev db.OutboxEvent, preloadedConfigs map[uuid.UUID]db.PostbackConfig) error {
	payload, err := parsePostbackPayload(ev.Payload)
	if err != nil {
		return err
	}

	idempotencyHash := postbackIdempotencyHash(payload)

	q := db.New(w.pool)
	slot, err := w.resolveDispatchSlot(ctx, q, idempotencyHash, payload)
	if err != nil {
		return err
	}
	if slot == dispatchSlotDuplicate {
		recordDuplicate()
		return ErrDuplicateEvent
	}

	if slot == dispatchSlotDelivered {
		if err := w.finalizeDispatchSent(ctx, q, idempotencyHash); err != nil {
			return ErrDispatchFinalizePending
		}
		return nil
	}

	var config db.PostbackConfig
	if preloadedConfigs != nil {
		cfg, ok := preloadedConfigs[payload.CampaignID]
		if !ok {
			slog.Warn("No postback config found for campaign, marking processed", "campaign_id", payload.CampaignID)
			return nil
		}
		config = cfg
	} else {
		var err error
		config, err = q.GetPostbackConfig(ctx, pgtype.UUID{Bytes: payload.CampaignID, Valid: true})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				slog.Warn("No postback config found for campaign, marking processed", "campaign_id", payload.CampaignID)
				return nil
			}
			return fmt.Errorf("failed to get postback config: %w", err)
		}
	}

	provider := strings.ToLower(config.Provider)
	if strings.TrimSpace(config.TestEventCode) != "" {
		payload.TestEventCode = strings.TrimSpace(config.TestEventCode)
	}

	var apiTokenDecrypted string
	if len(config.ApiTokenEncrypted) > 0 {
		decrypted, err := DecryptAESGCM(config.ApiTokenEncrypted, w.encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt API token: %w", err)
		}
		apiTokenDecrypted = string(decrypted)
	}

	limiter := w.getLimiter(config.UrlTemplate, config.Provider)
	if err := limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter wait aborted: %w", err)
	}

	adapter, ok := w.adapters[provider]
	if !ok {
		return fmt.Errorf("unsupported provider: %s", config.Provider)
	}

	if w.onDispatchAttempt != nil {
		w.onDispatchAttempt()
	}

	started := time.Now()
	err = w.dispatchWithRetry(ctx, adapter, &payload, config.UrlTemplate, apiTokenDecrypted, func() error {
		return w.markDispatchDelivered(ctx, q, idempotencyHash)
	})
	elapsed := time.Since(started).Seconds()
	if err != nil {
		recordDispatch(provider, "fail", elapsed)
		slog.Error("Postback dispatch failed completely, moving to DLQ", "error", err, "payload", payload)
		_, dlqErr := q.InsertPostbackDLQ(ctx, db.InsertPostbackDLQParams{
			OutboxEventID: ev.ID,
			CampaignID:    pgtype.UUID{Bytes: payload.CampaignID, Valid: true},
			ClickID:       payload.ClickID,
			EventType:     payload.EventType,
			Payload:       ev.Payload,
			FailuresCount: 5,
			LastError:     pgtype.Text{String: err.Error(), Valid: true},
			Status:        "FAILED",
		})
		if dlqErr != nil {
			slog.Error("Failed to insert into DLQ", "error", dlqErr)
			return fmt.Errorf("original error: %w; dlq insert failed: %w", err, dlqErr)
		}
		recordDLQ(provider)
		if markErr := q.UpdatePostbackDispatchStatus(ctx, db.UpdatePostbackDispatchStatusParams{
			IdempotencyHash: idempotencyHash,
			Status:          postbackDispatchStatusFailed,
			ErrorMessage:    pgtype.Text{String: err.Error(), Valid: true},
			Status_2:        postbackDispatchStatusInFlight,
		}); markErr != nil {
			slog.Warn("Failed to mark dispatch failed", "error", markErr)
		}
		return fmt.Errorf("dispatch failed (moved to DLQ): %w", err)
	}

	recordDispatch(provider, "success", elapsed)

	if err := w.finalizeDispatchSent(ctx, q, idempotencyHash); err != nil {
		return ErrDispatchFinalizePending
	}
	return nil
}

func (w *PostbackWorker) dispatchWithRetry(ctx context.Context, adapter PostbackAdapter, payload *PostbackPayload, urlTemplate, token string, onDelivered func() error) error {
	var lastErr error
	maxRetries := 5

	for attempt := range maxRetries {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * 200 * time.Millisecond
			jitter := time.Duration(randInt64(50)) * time.Millisecond
			sleepTime := backoff + jitter

			slog.Info("Retrying postback dispatch", "attempt", attempt, "sleep", sleepTime)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(sleepTime):
			}
		}

		err := adapter.Send(ctx, w.client, payload, urlTemplate, token)
		if err == nil {
			if onDelivered != nil {
				if err := onDelivered(); err != nil {
					return err
				}
			}
			return nil
		}
		lastErr = err

		slog.Warn("Postback dispatch attempt failed", "attempt", attempt+1, "error", err)
	}

	return fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

func (w *PostbackWorker) getLimiter(targetURL string, provider string) *rate.Limiter {
	key := provider
	if u, err := url.Parse(targetURL); err == nil && u.Host != "" {
		key = u.Host
	}

	w.limitersMu.Lock()
	defer w.limitersMu.Unlock()

	lim, exists := w.limiters[key]
	if !exists {
		lim = rate.NewLimiter(rate.Limit(100), 200)
		w.limiters[key] = lim
	}
	return lim
}

func randInt64(upper int64) int64 {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	val := int64(b[0]) | int64(b[1])<<8 | int64(b[2])<<16 | int64(b[3])<<24 | int64(b[4])<<32 | int64(b[5])<<40 | int64(b[6])<<48 | int64(b[7])<<56
	if val < 0 {
		val = -val
	}
	return val % upper
}

func EncryptAESGCM(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func DecryptAESGCM(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, actualCiphertext, nil)
}

type PostbackAdapter interface {
	Send(ctx context.Context, client *http.Client, payload *PostbackPayload, urlTemplate string, apiTokenDecrypted string) error
}
