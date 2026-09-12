package postback

import (
	"context"
	"encoding/json"
	"testing"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

type multiOutboundStore struct {
	outbounds []db.CampaignOutboundPostback
	legacy    map[uuid.UUID]db.PostbackConfig
	payloads  [][]byte
	notBefore []pgtype.Timestamptz
}

func (s *multiOutboundStore) ListOutboundPostbacksByCampaignIDs(_ context.Context, ids []pgtype.UUID) ([]db.CampaignOutboundPostback, error) {
	if s == nil || len(s.outbounds) == 0 {
		return nil, nil
	}
	out := make([]db.CampaignOutboundPostback, 0, len(s.outbounds))
	for _, row := range s.outbounds {
		for _, id := range ids {
			if row.CampaignID.Valid && id.Bytes == row.CampaignID.Bytes {
				out = append(out, row)
			}
		}
	}
	return out, nil
}

func (s *multiOutboundStore) ListPostbackConfigsByCampaignIDs(_ context.Context, ids []pgtype.UUID) ([]db.PostbackConfig, error) {
	if s == nil || len(s.legacy) == 0 {
		return nil, nil
	}
	out := make([]db.PostbackConfig, 0, len(ids))
	for _, id := range ids {
		if cfg, ok := s.legacy[uuid.UUID(id.Bytes)]; ok {
			out = append(out, cfg)
		}
	}
	return out, nil
}

func (s *multiOutboundStore) ListCampaignsByIDs(_ context.Context, ids []pgtype.UUID) ([]db.Campaign, error) {
	out := make([]db.Campaign, 0, len(ids))
	for _, id := range ids {
		out = append(out, db.Campaign{
			ID:         id,
			CustomerID: pgtype.UUID{Bytes: benchCustomerID, Valid: true},
		})
	}
	return out, nil
}

func (s *multiOutboundStore) CreatePostbackOutboxEventsBatch(_ context.Context, arg db.CreatePostbackOutboxEventsBatchParams) error {
	if s != nil {
		s.payloads = append(s.payloads, arg.Payloads...)
		s.notBefore = append(s.notBefore, arg.NotBefore...)
	}
	return nil
}

func outboundRow(campID, postbackID uuid.UUID, triggerKind, triggerValue, url string) db.CampaignOutboundPostback {
	return outboundRowWithSample(campID, postbackID, triggerKind, triggerValue, url, 100)
}

func outboundRowWithSample(campID, postbackID uuid.UUID, triggerKind, triggerValue, url string, samplePercent int32) db.CampaignOutboundPostback {
	return outboundRowWithSampleDelay(campID, postbackID, triggerKind, triggerValue, url, samplePercent, 0)
}

func outboundRowWithSampleDelay(campID, postbackID uuid.UUID, triggerKind, triggerValue, url string, samplePercent, delaySeconds int32) db.CampaignOutboundPostback {
	return db.CampaignOutboundPostback{
		ID:            pgtype.UUID{Bytes: postbackID, Valid: true},
		CampaignID:    pgtype.UUID{Bytes: campID, Valid: true},
		Provider:      "webhook",
		UrlTemplate:   url,
		TargetEvent:   "conversion",
		TriggerKind:   triggerKind,
		TriggerValue:  triggerValue,
		Enabled:       true,
		SamplePercent: samplePercent,
		DelaySeconds:  delaySeconds,
	}
}

func TestMultiOutbound_enqueuesMatchingRows(t *testing.T) {
	campID := uuid.New()
	pbA := uuid.New()
	pbB := uuid.New()
	store := &multiOutboundStore{
		outbounds: []db.CampaignOutboundPostback{
			outboundRow(campID, pbA, OutboundTriggerGoal, "approved", "https://a.example/postback"),
			outboundRow(campID, pbB, OutboundTriggerGoal, "hold", "https://b.example/postback"),
		},
	}
	enq := NewConversionPostbackEnqueuer(store)
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		ClickID:    "clk-1",
		Payload:    []byte(`{"goal_name":"approved"}`),
	}
	enq.OnBatchStored(context.Background(), []*domain.Event{evt})
	require.Len(t, store.payloads, 1)
	var payload PostbackPayload
	require.NoError(t, json.Unmarshal(store.payloads[0], &payload))
	require.Equal(t, pbA, payload.OutboundPostbackID)
}

func TestMultiOutbound_idempotencyDistinctPerPostback_holdout(t *testing.T) {
	customerID := uuid.New()
	campID := uuid.New()
	pbA := uuid.New()
	pbB := uuid.New()
	hashA := postbackIdempotencyHash(PostbackPayload{
		CustomerID:         customerID,
		CampaignID:         campID,
		ClickID:            "clk-1",
		EventType:          "conversion",
		OutboundPostbackID: pbA,
	})
	hashB := postbackIdempotencyHash(PostbackPayload{
		CustomerID:         customerID,
		CampaignID:         campID,
		ClickID:            "clk-1",
		EventType:          "conversion",
		OutboundPostbackID: pbB,
	})
	require.NotEqual(t, hashA, hashB)
	legacy := postbackIdempotencyHash(PostbackPayload{
		CustomerID: customerID,
		CampaignID: campID,
		ClickID:    "clk-1",
		EventType:  "conversion",
	})
	require.NotEqual(t, legacy, hashA)
}

func TestMultiOutbound_legacyFallbackWhenNoRows(t *testing.T) {
	campID := benchCampaignID
	store := &multiOutboundStore{
		legacy: map[uuid.UUID]db.PostbackConfig{
			campID: {
				CampaignID:  pgtype.UUID{Bytes: campID, Valid: true},
				Provider:    "webhook",
				UrlTemplate: "https://legacy.example/postback",
				TargetEvent: "conversion",
			},
		},
	}
	enq := NewConversionPostbackEnqueuer(store)
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		ClickID:    "clk-legacy",
		Payload:    []byte(`{"goal_name":"lead"}`),
	}
	enq.OnBatchStored(context.Background(), []*domain.Event{evt})
	require.Len(t, store.payloads, 1)
	var payload PostbackPayload
	require.NoError(t, json.Unmarshal(store.payloads[0], &payload))
	require.Equal(t, uuid.Nil, payload.OutboundPostbackID)
}

func TestMultiOutbound_samplePercent_holdout(t *testing.T) {
	campID := uuid.New()
	pbA := uuid.New()
	store := &multiOutboundStore{
		outbounds: []db.CampaignOutboundPostback{
			outboundRowWithSample(campID, pbA, OutboundTriggerConversion, "", "https://sample.example/postback", 0),
		},
	}
	enq := NewConversionPostbackEnqueuer(store)
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		ClickID:    "clk-sample-0",
	}
	enq.OnBatchStored(context.Background(), []*domain.Event{evt})
	require.Empty(t, store.payloads)

	store.payloads = nil
	store.outbounds[0].SamplePercent = 100
	enq.OnBatchStored(context.Background(), []*domain.Event{evt})
	require.Len(t, store.payloads, 1)
}

func TestDelayedPostback_enqueueSetsNotBefore_holdout(t *testing.T) {
	campID := uuid.New()
	pbA := uuid.New()
	store := &multiOutboundStore{
		outbounds: []db.CampaignOutboundPostback{
			outboundRowWithSampleDelay(campID, pbA, OutboundTriggerConversion, "", "https://delay.example/postback", 100, 120),
		},
	}
	enq := NewConversionPostbackEnqueuer(store)
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		ClickID:    "clk-delay-1",
	}
	enq.OnBatchStored(context.Background(), []*domain.Event{evt})
	require.Len(t, store.payloads, 1)
	require.Len(t, store.notBefore, 1)
	require.True(t, store.notBefore[0].Valid)
}

func TestDelayedPostback_zeroDelayLeavesNotBeforeNull_holdout(t *testing.T) {
	campID := uuid.New()
	pbA := uuid.New()
	store := &multiOutboundStore{
		outbounds: []db.CampaignOutboundPostback{
			outboundRowWithSampleDelay(campID, pbA, OutboundTriggerConversion, "", "https://immediate.example/postback", 100, 0),
		},
	}
	enq := NewConversionPostbackEnqueuer(store)
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		ClickID:    "clk-delay-0",
	}
	enq.OnBatchStored(context.Background(), []*domain.Event{evt})
	require.Len(t, store.payloads, 1)
	require.Len(t, store.notBefore, 1)
	require.False(t, store.notBefore[0].Valid)
}

func TestOutboundPostbackMatchesEvent_statusTrigger(t *testing.T) {
	row := outboundRow(uuid.New(), uuid.New(), OutboundTriggerStatus, "hold", "https://example/postback")
	evt := &domain.Event{Type: "conversion", Payload: []byte(`{"status":"hold"}`)}
	require.True(t, OutboundPostbackMatchesEvent(row, evt))
	evt.Payload = []byte(`{"status":"sale"}`)
	require.False(t, OutboundPostbackMatchesEvent(row, evt))
}
