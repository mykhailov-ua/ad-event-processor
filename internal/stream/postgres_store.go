package stream

import (
	"context"
	"sync"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/piihash"

	"github.com/jackc/pgx/v5/pgtype"
)

type postgresBatchArrays struct {
	clickIDs     []string
	campaignIDs  []pgtype.UUID
	userIDs      []string
	eventTypes   []string
	payloads     [][]byte
	ipAddresses  []string
	ipHashes     [][]byte
	userAgents   []string
	createdAts   []pgtype.Timestamptz
	createdDates []pgtype.Date
}

var postgresBatchArraysPool = sync.Pool{
	New: func() any {
		return &postgresBatchArrays{
			clickIDs:     make([]string, 0, 1000),
			campaignIDs:  make([]pgtype.UUID, 0, 1000),
			userIDs:      make([]string, 0, 1000),
			eventTypes:   make([]string, 0, 1000),
			payloads:     make([][]byte, 0, 1000),
			ipAddresses:  make([]string, 0, 1000),
			ipHashes:     make([][]byte, 0, 1000),
			userAgents:   make([]string, 0, 1000),
			createdAts:   make([]pgtype.Timestamptz, 0, 1000),
			createdDates: make([]pgtype.Date, 0, 1000),
		}
	},
}

type PostgresStore struct {
	queries        db.Querier
	writeTimeout   time.Duration
	postgresGate   *ProcessorPostgresGate
	hashIPAtInsert bool
	piiHasher      *piihash.Hasher
}

func NewPostgresStore(queries db.Querier, writeTimeout time.Duration) *PostgresStore {
	return &PostgresStore{
		queries:      queries,
		writeTimeout: writeTimeout,
	}
}

func NewPostgresStoreWithGate(queries db.Querier, writeTimeout time.Duration, gate *ProcessorPostgresGate) *PostgresStore {
	return &PostgresStore{
		queries:      queries,
		writeTimeout: writeTimeout,
		postgresGate: gate,
	}
}

func (s *PostgresStore) SetQuerier(queries db.Querier) {
	if s != nil && queries != nil {
		s.queries = queries
	}
}

func (s *PostgresStore) StoreBatch(ctx context.Context, events []*domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	if s.postgresGate != nil {
		if err := s.postgresGate.Acquire(ctx); err != nil {
			return err
		}
		defer s.postgresGate.Release()
	}

	arrs := postgresBatchArraysPool.Get().(*postgresBatchArrays)
	defer func() {
		for i := range arrs.clickIDs {
			arrs.clickIDs[i] = ""
		}
		arrs.clickIDs = arrs.clickIDs[:0]

		for i := range arrs.campaignIDs {
			arrs.campaignIDs[i] = pgtype.UUID{}
		}
		arrs.campaignIDs = arrs.campaignIDs[:0]

		for i := range arrs.userIDs {
			arrs.userIDs[i] = ""
		}
		arrs.userIDs = arrs.userIDs[:0]

		for i := range arrs.eventTypes {
			arrs.eventTypes[i] = ""
		}
		arrs.eventTypes = arrs.eventTypes[:0]

		for i := range arrs.payloads {
			arrs.payloads[i] = nil
		}
		arrs.payloads = arrs.payloads[:0]

		for i := range arrs.ipAddresses {
			arrs.ipAddresses[i] = ""
		}
		arrs.ipAddresses = arrs.ipAddresses[:0]

		for i := range arrs.ipHashes {
			arrs.ipHashes[i] = nil
		}
		arrs.ipHashes = arrs.ipHashes[:0]

		for i := range arrs.userAgents {
			arrs.userAgents[i] = ""
		}
		arrs.userAgents = arrs.userAgents[:0]

		for i := range arrs.createdAts {
			arrs.createdAts[i] = pgtype.Timestamptz{}
		}
		arrs.createdAts = arrs.createdAts[:0]

		for i := range arrs.createdDates {
			arrs.createdDates[i] = pgtype.Date{}
		}
		arrs.createdDates = arrs.createdDates[:0]

		postgresBatchArraysPool.Put(arrs)
	}()

	n := len(events)
	if cap(arrs.clickIDs) < n {
		arrs.clickIDs = make([]string, 0, n)
		arrs.campaignIDs = make([]pgtype.UUID, 0, n)
		arrs.userIDs = make([]string, 0, n)
		arrs.eventTypes = make([]string, 0, n)
		arrs.payloads = make([][]byte, 0, n)
		arrs.ipAddresses = make([]string, 0, n)
		arrs.ipHashes = make([][]byte, 0, n)
		arrs.userAgents = make([]string, 0, n)
		arrs.createdAts = make([]pgtype.Timestamptz, 0, n)
		arrs.createdDates = make([]pgtype.Date, 0, n)
	}

	defaultPayload := []byte("{}")

	for _, evt := range events {
		arrs.clickIDs = append(arrs.clickIDs, evt.ClickID)
		arrs.campaignIDs = append(arrs.campaignIDs, pgtype.UUID{Bytes: evt.CampaignID, Valid: true})
		arrs.userIDs = append(arrs.userIDs, evt.UserID)
		arrs.eventTypes = append(arrs.eventTypes, evt.Type)
		if len(evt.Payload) == 0 {
			arrs.payloads = append(arrs.payloads, defaultPayload)
		} else {
			arrs.payloads = append(arrs.payloads, evt.Payload)
		}
		if s.hashIPAtInsert && s.piiHasher != nil && evt.IP != "" {
			h := s.piiHasher.HashIP(evt.IP)
			arrs.ipHashes = append(arrs.ipHashes, h[:])
			arrs.ipAddresses = append(arrs.ipAddresses, "")
		} else {
			arrs.ipHashes = append(arrs.ipHashes, nil)
			arrs.ipAddresses = append(arrs.ipAddresses, evt.IP)
		}
		arrs.userAgents = append(arrs.userAgents, evt.UA)

		const secondsPerDay = 86400
		unix := evt.CreatedAt.Unix()
		midnight := (unix / secondsPerDay) * secondsPerDay
		arrs.createdAts = append(arrs.createdAts, pgtype.Timestamptz{Time: evt.CreatedAt, Valid: true})
		arrs.createdDates = append(arrs.createdDates, pgtype.Date{
			Time:  time.Unix(midnight, 0).UTC(),
			Valid: true,
		})
	}

	var err error
	waitTime := InitialWait

	for i := 0; i <= MaxRetries; i++ {
		dbCtx, cancel := context.WithTimeout(ctx, s.writeTimeout)
		start := time.Now()

		err = s.queries.InsertEventsBatch(dbCtx, db.InsertEventsBatchParams{
			ClickIds:     arrs.clickIDs,
			CampaignIds:  arrs.campaignIDs,
			UserIds:      arrs.userIDs,
			EventTypes:   arrs.eventTypes,
			Payloads:     arrs.payloads,
			IpAddresses:  arrs.ipAddresses,
			IpHashes:     arrs.ipHashes,
			UserAgents:   arrs.userAgents,
			CreatedAt:    arrs.createdAts,
			CreatedDates: arrs.createdDates,
		})

		duration := time.Since(start).Seconds()
		cancel()

		if err == nil {
			metrics.DBWriteDuration.WithLabelValues("postgres").Observe(duration)
			return nil
		}

		if i < MaxRetries {
			timer := time.NewTimer(waitTime)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
				waitTime *= 2
				if waitTime > MaxWait {
					waitTime = MaxWait
				}
			}
		}
	}

	metrics.DBWriteErrors.WithLabelValues("postgres").Inc()
	return err
}

func (s *PostgresStore) StoreStatsBatch(ctx context.Context, events []*domain.Event) error {
	rollup := rollupCampaignStats(events)
	if len(rollup) == 0 {
		return nil
	}

	if s.postgresGate != nil {
		if err := s.postgresGate.Acquire(ctx); err != nil {
			return err
		}
		defer s.postgresGate.Release()
	}

	campaignIDs, impressions, clicks, conversions := campaignStatRollupArrays(rollup)

	var err error
	waitTime := InitialWait
	for i := 0; i <= MaxRetries; i++ {
		dbCtx, cancel := context.WithTimeout(ctx, s.writeTimeout)
		start := time.Now()
		err = s.queries.UpdateCampaignStatsBatch(dbCtx, db.UpdateCampaignStatsBatchParams{
			CampaignIds: campaignIDs,
			Impressions: impressions,
			Clicks:      clicks,
			Conversions: conversions,
		})
		duration := time.Since(start).Seconds()
		cancel()

		if err == nil {
			metrics.DBWriteDuration.WithLabelValues("postgres_stats").Observe(duration)
			metrics.SettlementStatsCampaignsFlushed.Add(float64(len(campaignIDs)))
			return nil
		}

		if i < MaxRetries {
			timer := time.NewTimer(waitTime)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
				waitTime *= 2
				if waitTime > MaxWait {
					waitTime = MaxWait
				}
			}
		}
	}

	metrics.DBWriteErrors.WithLabelValues("postgres_stats").Inc()
	return err
}

func (s *PostgresStore) SetPIIHashAtInsert(h *piihash.Hasher) {
	if h == nil {
		return
	}
	s.piiHasher = h
	s.hashIPAtInsert = true
}

func (s *PostgresStore) Close() error {
	return nil
}
