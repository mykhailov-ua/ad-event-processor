package privacyadmin

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrConsentInvalidSignature = errors.New("invalid consent signature")
	ErrConsentInvalidPayload   = errors.New("invalid consent payload")
)

func VerifyConsentHMAC(secret []byte, body []byte, signatureHex string) error {
	if len(secret) == 0 {
		return ErrConsentInvalidSignature
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	expected := mac.Sum(nil)
	got, err := hex.DecodeString(signatureHex)
	if err != nil {
		return ErrConsentInvalidSignature
	}
	if !hmac.Equal(expected, got) {
		return ErrConsentInvalidSignature
	}
	return nil
}

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

func (st *Store) RecordConsent(ctx context.Context, in ConsentRecord) error {
	if in.UserID == "" {
		return st.host.ErrValidation("user_id is required")
	}
	if in.Source == "" {
		return st.host.ErrValidation("source is required")
	}
	if in.Purposes < 0 {
		return st.host.ErrValidation("purposes must be non-negative")
	}

	hash := domain.HashUserID(in.UserID)
	adStorage, analyticsStorage := domain.ConsentFlagsFromPurposes(in.Purposes)

	return pgx.BeginFunc(ctx, st.poolOrNil(), func(tx pgx.Tx) error {
		q := db.New(tx)
		if err := q.InsertConsentEvent(ctx, db.InsertConsentEventParams{
			UserIDHash: hash,
			Purposes:   in.Purposes,
			Source:     in.Source,
		}); err != nil {
			return fmt.Errorf("insert consent event: %w", err)
		}
		if err := q.UpsertUserConsentState(ctx, db.UpsertUserConsentStateParams{
			UserIDHash:       hash,
			AdStorage:        adStorage,
			AnalyticsStorage: analyticsStorage,
			Purposes:         in.Purposes,
		}); err != nil {
			return fmt.Errorf("upsert consent state: %w", err)
		}
		payload, err := coldpath.MarshalOutbox(userConsentOutboxPayload{
			UserIDHash: hex.EncodeToString(hash),
			Purposes:   in.Purposes,
		})
		if err != nil {
			return fmt.Errorf("marshal consent outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "SYNC_USER_CONSENT",
			Payload:   payload,
		})
		return err
	})
}

func (st *Store) UpdateCampaignConsentRequirements(ctx context.Context, campaignID uuid.UUID, purposes int16) error {
	if purposes < 0 {
		return st.host.ErrValidation("require_consent_purposes must be non-negative")
	}
	return pgx.BeginFunc(ctx, st.poolOrNil(), func(tx pgx.Tx) error {
		q := db.New(tx)
		if _, err := q.UpdateCampaignConsentPurposes(ctx, db.UpdateCampaignConsentPurposesParams{
			ID:                     domain.ToUUID(campaignID),
			RequireConsentPurposes: purposes,
		}); err != nil {
			return st.host.MapCampaignNotFound(err)
		}
		payload, err := coldpath.MarshalOutbox(campaignConsentOutboxPayload{CampaignID: campaignID.String()})
		if err != nil {
			return err
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "UPDATE_CAMPAIGN_CONSENT",
			Payload:   payload,
		})
		return err
	})
}

func (st *Store) CleanupConsentEvents(ctx context.Context) error {
	months := st.host.ConsentRetentionMonths()
	if months <= 0 {
		return nil
	}
	threshold := time.Now().AddDate(0, -months, 0)
	return db.New(st.poolOrNil()).CleanupConsentEventsOlderThan(ctx, pgtype.Timestamptz{Time: threshold, Valid: true})
}

func (st *Store) CreatePrivacyErasureRequest(ctx context.Context, userID string) (uuid.UUID, error) {
	if userID == "" {
		return uuid.Nil, st.host.ErrValidation("user_id is required")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}
	hash := domain.HashUserID(userID)
	_, err = db.New(st.poolOrNil()).CreatePrivacyErasureRequest(ctx, db.CreatePrivacyErasureRequestParams{
		ID:            domain.ToUUID(id),
		UserIDHash:    hash,
		SubjectUserID: userID,
	})
	return id, err
}

func (st *Store) ProcessPrivacyErasureTick(ctx context.Context) error {
	opCtx, cancel := st.host.WorkerBatchContext(ctx)
	defer cancel()

	q := db.New(st.poolOrNil())
	rows, err := q.ListPrivacyErasureRequestsByStatus(opCtx, db.ListPrivacyErasureRequestsByStatusParams{
		Status: db.PrivacyErasureStatusPENDING,
		Limit:  20,
	})
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := st.advanceErasurePG(opCtx, row); err != nil {
			if failErr := st.failErasure(opCtx, row.ID, err); failErr != nil {
				return fmt.Errorf("privacy erasure: mark pg anonymize failed: %w (cause: %w)", failErr, err)
			}
		}
	}

	rows, err = q.ListPrivacyErasureRequestsByStatus(opCtx, db.ListPrivacyErasureRequestsByStatusParams{
		Status: db.PrivacyErasureStatusPGANONYMIZED,
		Limit:  20,
	})
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := st.enqueueErasureRedisPurge(opCtx, row); err != nil {
			if failErr := st.failErasure(opCtx, row.ID, err); failErr != nil {
				return fmt.Errorf("privacy erasure: mark redis purge failed: %w (cause: %w)", failErr, err)
			}
		}
	}

	rows, err = q.ListPrivacyErasureRequestsByStatus(opCtx, db.ListPrivacyErasureRequestsByStatusParams{
		Status: db.PrivacyErasureStatusREDISPURGED,
		Limit:  20,
	})
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := st.advanceErasureCH(opCtx, row); err != nil {
			if failErr := st.failErasure(opCtx, row.ID, err); failErr != nil {
				return fmt.Errorf("privacy erasure: mark clickhouse anonymize failed: %w (cause: %w)", failErr, err)
			}
		}
	}
	return nil
}

func (st *Store) advanceErasurePG(ctx context.Context, row db.PrivacyErasureRequest) error {
	return pgx.BeginFunc(ctx, st.poolOrNil(), func(tx pgx.Tx) error {
		q := db.New(tx)
		locked, err := q.GetPrivacyErasureRequestForUpdate(ctx, row.ID)
		if err != nil {
			return err
		}
		if locked.Status != db.PrivacyErasureStatusPENDING {
			return nil
		}
		if locked.SubjectUserID != "" {
			if err := q.AnonymizeEventsByUserID(ctx, pgtype.Text{String: locked.SubjectUserID, Valid: true}); err != nil {
				return err
			}
		}
		if err := q.AnonymizeConsentEventsByUserHash(ctx, locked.UserIDHash); err != nil {
			return err
		}
		if err := q.DeleteUserConsentState(ctx, locked.UserIDHash); err != nil {
			return err
		}
		return q.UpdatePrivacyErasureStatus(ctx, db.UpdatePrivacyErasureStatusParams{
			ID:     locked.ID,
			Status: db.PrivacyErasureStatusPGANONYMIZED,
		})
	})
}

func (st *Store) enqueueErasureRedisPurge(ctx context.Context, row db.PrivacyErasureRequest) error {
	return pgx.BeginFunc(ctx, st.poolOrNil(), func(tx pgx.Tx) error {
		q := db.New(tx)
		locked, err := q.GetPrivacyErasureRequestForUpdate(ctx, row.ID)
		if err != nil {
			return err
		}
		if locked.Status != db.PrivacyErasureStatusPGANONYMIZED {
			return nil
		}
		if locked.LastError.Valid && locked.LastError.String == "purge_enqueued" {
			return nil
		}
		payload, err := coldpath.MarshalOutbox(purgeUserDataOutboxPayload{
			ErasureID:     uuid.UUID(locked.ID.Bytes).String(),
			UserIDHash:    hex.EncodeToString(locked.UserIDHash),
			SubjectUserID: locked.SubjectUserID,
		})
		if err != nil {
			return err
		}
		if _, err := q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "PURGE_USER_DATA",
			Payload:   payload,
		}); err != nil {
			return err
		}
		return q.UpdatePrivacyErasureStatus(ctx, db.UpdatePrivacyErasureStatusParams{
			ID:        locked.ID,
			Status:    db.PrivacyErasureStatusPGANONYMIZED,
			LastError: pgtype.Text{String: "purge_enqueued", Valid: true},
		})
	})
}

func (st *Store) advanceErasureCH(ctx context.Context, row db.PrivacyErasureRequest) error {
	userID := row.SubjectUserID
	if userID != "" {
		if err := st.host.ClickHouseDeleteFraudEventsByUser(ctx, userID); err != nil {
			return st.failErasure(ctx, row.ID, err)
		}
	}
	return pgx.BeginFunc(ctx, st.poolOrNil(), func(tx pgx.Tx) error {
		q := db.New(tx)
		if err := q.UpdatePrivacyErasureStatus(ctx, db.UpdatePrivacyErasureStatusParams{
			ID:     row.ID,
			Status: db.PrivacyErasureStatusCHPURGED,
		}); err != nil {
			return err
		}
		if err := q.UpdatePrivacyErasureStatus(ctx, db.UpdatePrivacyErasureStatusParams{
			ID:     row.ID,
			Status: db.PrivacyErasureStatusCOMPLETED,
		}); err != nil {
			return err
		}
		return q.ClearErasureSubjectUserID(ctx, row.ID)
	})
}

func (st *Store) failErasure(ctx context.Context, id pgtype.UUID, err error) error {
	msg := err.Error()
	return db.New(st.poolOrNil()).UpdatePrivacyErasureStatus(ctx, db.UpdatePrivacyErasureStatusParams{
		ID:        id,
		Status:    db.PrivacyErasureStatusFAILED,
		LastError: pgtype.Text{String: msg, Valid: true},
	})
}

func (st *Store) PurgeUserDataRedis(ctx context.Context, hashHex, subjectUserID string) error {
	if err := st.host.PurgeUserRedisKeys(ctx, hashHex, subjectUserID); err != nil {
		return err
	}
	return st.host.PublishConsentUpdate(ctx, hashHex)
}

func (st *Store) SyncUserConsentToRedis(ctx context.Context, hashHex string, purposes int16) error {
	return st.host.SyncConsentRedisKey(ctx, hashHex, purposes)
}

func (st *Store) MarkErasureRedisPurgeDone(ctx context.Context, erasureID uuid.UUID, partialErr error) error {
	status := db.PrivacyErasureStatusREDISPURGED
	if partialErr != nil {
		return db.New(st.poolOrNil()).UpdatePrivacyErasureStatus(ctx, db.UpdatePrivacyErasureStatusParams{
			ID:        domain.ToUUID(erasureID),
			Status:    db.PrivacyErasureStatusFAILED,
			LastError: pgtype.Text{String: partialErr.Error(), Valid: true},
		})
	}
	return db.New(st.poolOrNil()).UpdatePrivacyErasureStatus(ctx, db.UpdatePrivacyErasureStatusParams{
		ID:     domain.ToUUID(erasureID),
		Status: status,
	})
}

func (st *Store) ConsentUpdateChannel() string {
	return st.host.ConsentUpdateChannel()
}
