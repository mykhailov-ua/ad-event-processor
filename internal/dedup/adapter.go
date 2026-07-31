package dedup

import (
	"context"
	"errors"
	"time"

	db "espx/internal/domain/db"
	"espx/internal/metrics"
	"espx/pkg/dedupkey"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Outcome string

const (
	OutcomeConfirmed        Outcome = "confirmed"
	OutcomeAlreadyConfirmed Outcome = "already_confirmed"
	OutcomeHashMismatch     Outcome = "hash_mismatch"
	OutcomeRejected         Outcome = "rejected"
	OutcomePending          Outcome = "pending"
)

type ClaimResult struct {
	Outcome  Outcome
	FactorD  uuid.UUID
	DedupKey string
	Scope    dedupkey.Scope
	FactorU  uuid.UUID
}

type Adapter struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	region  uint8
	epoch   uint32
}

func NewAdapter(pool *pgxpool.Pool, regionCode uint8, sourceEpoch uint32) *Adapter {
	if pool == nil {
		return nil
	}
	return &Adapter{
		pool:    pool,
		queries: db.New(pool),
		region:  regionCode,
		epoch:   sourceEpoch,
	}
}

func (a *Adapter) SourceEpoch() uint32 {
	if a == nil {
		return 0
	}
	return a.epoch
}

func (a *Adapter) RegionScope(sourceID uuid.UUID, seqStart, seqEnd int64) dedupkey.Scope {
	regionID := dedupkey.RegionUUID(a.region)
	if a.region == 0 {
		regionID = dedupkey.RegionUUID(0)
	}
	return dedupkey.Scope{
		RegionID:    regionID,
		SourceID:    sourceID,
		SourceEpoch: a.epoch,
		SeqStart:    seqStart,
		SeqEnd:      seqEnd,
	}
}

func (a *Adapter) ClaimConfirm(ctx context.Context, scope dedupkey.Scope, factorU uuid.UUID) (ClaimResult, error) {
	if a == nil || a.pool == nil {
		key := dedupkey.FormatCanonical(scope, factorU, uuid.Nil)
		return ClaimResult{
			Outcome:  OutcomeConfirmed,
			DedupKey: key,
			Scope:    scope,
			FactorU:  factorU,
		}, nil
	}

	start := time.Now()
	var outcome string
	var factorD pgtype.UUID
	var dedupKey string
	err := a.pool.QueryRow(ctx, `
		SELECT outcome, factor_d, dedup_key
		FROM dedup_claim_confirm($1, $2, $3, $4, $5, $6)`,
		scope.RegionID, scope.SourceID, int64(scope.SourceEpoch),
		scope.SeqStart, scope.SeqEnd, factorU,
	).Scan(&outcome, &factorD, &dedupKey)
	if err != nil {
		return ClaimResult{}, err
	}

	metrics.DedupProposalTotal.WithLabelValues(outcome).Inc()
	metrics.DedupConfirmLatency.Observe(time.Since(start).Seconds())

	if outcome == string(OutcomeHashMismatch) {
		metrics.DedupMismatchTotal.Inc()
	}

	var factorDUUID uuid.UUID
	if factorD.Valid {
		factorDUUID = uuid.UUID(factorD.Bytes)
	}

	return ClaimResult{
		Outcome:  Outcome(outcome),
		FactorD:  factorDUUID,
		DedupKey: dedupKey,
		Scope:    scope,
		FactorU:  factorU,
	}, nil
}

func (r ClaimResult) ShouldApply() bool {
	switch r.Outcome {
	case OutcomeConfirmed:
		return true
	case OutcomeAlreadyConfirmed:
		return true
	default:
		return false
	}
}

func (a *Adapter) RecordApply(ctx context.Context, dedupKey string) error {
	if a == nil || a.pool == nil || dedupKey == "" {
		return nil
	}
	_, err := a.pool.Exec(ctx, `INSERT INTO sync_idempotency (id) VALUES ($1) ON CONFLICT DO NOTHING`, dedupKey)
	return err
}

func (a *Adapter) NeedsResumeApply(ctx context.Context, dedupKey string) (bool, error) {
	if a == nil || dedupKey == "" {
		return true, nil
	}
	exists, err := a.queries.DedupSyncIdempotencyExists(ctx, dedupKey)
	if err != nil {
		return false, err
	}
	return !exists, nil
}

func (a *Adapter) RejectStaleProposals(ctx context.Context) (int64, error) {
	if a == nil {
		return 0, nil
	}
	return a.queries.RejectStaleDedupProposals(ctx)
}

var ErrHashMismatch = errors.New("dedup hash mismatch")

var ErrRejected = errors.New("dedup proposal rejected")

func GuardOutcome(result ClaimResult) error {
	switch result.Outcome {
	case OutcomeHashMismatch:
		return ErrHashMismatch
	case OutcomeRejected:
		return ErrRejected
	case OutcomePending:
		return errors.New("dedup proposal pending")
	default:
		return nil
	}
}
