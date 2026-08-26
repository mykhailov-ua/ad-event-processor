package postback

import (
	"context"
	"strings"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

type stubConversionClickStore struct {
	clicks  map[string]clickSnapshot
	goals   map[conversionGoalKey]struct{}
	loadErr error
}

func (s *stubConversionClickStore) LoadClicks(_ context.Context, clickIDs []string) (map[string]clickSnapshot, error) {
	if s.loadErr != nil {
		return nil, s.loadErr
	}
	if s.clicks == nil {
		return nil, nil
	}
	out := make(map[string]clickSnapshot, len(clickIDs))
	for _, id := range clickIDs {
		if snap, ok := s.clicks[id]; ok {
			out[id] = snap
		}
	}
	return out, nil
}

func (s *stubConversionClickStore) LoadExistingGoals(_ context.Context, keys []conversionGoalKey) (map[conversionGoalKey]struct{}, error) {
	if s.loadErr != nil {
		return nil, s.loadErr
	}
	if s.goals == nil {
		return nil, nil
	}
	out := make(map[conversionGoalKey]struct{}, len(keys))
	for _, key := range keys {
		if _, ok := s.goals[key]; ok {
			out[key] = struct{}{}
		}
	}
	return out, nil
}

type stubConversionCampaignReader struct {
	silent      map[uuid.UUID]bool
	rejectRules map[uuid.UUID]domain.ConversionRejectRules
}

func (s *stubConversionCampaignReader) ListSilentRejectByCampaignIDs(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	if s == nil || s.silent == nil {
		return nil, nil
	}
	out := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		if v, ok := s.silent[id]; ok {
			out[id] = v
		}
	}
	return out, nil
}

func (s *stubConversionCampaignReader) ListConversionRejectRulesByCampaignIDs(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]domain.ConversionRejectRules, error) {
	if s == nil || s.rejectRules == nil {
		return nil, nil
	}
	out := make(map[uuid.UUID]domain.ConversionRejectRules, len(ids))
	for _, id := range ids {
		if v, ok := s.rejectRules[id]; ok {
			out[id] = v
		}
	}
	return out, nil
}

func conversionRejectCfg() config.ConversionReject {
	return config.ConversionReject{
		Enabled:          true,
		MinTTCDurationMs: 3000,
		RejectNoClick:    true,
		RejectLowTTC:     true,
		RejectDuplicate:  true,
		RejectIPDrift:    true,
	}
}

func TestConversionReject_NoClickStore_defersPendingValidation(t *testing.T) {
	campID := uuid.New()
	applier := NewConversionRejectApplier(conversionRejectCfg(), nil, nil, nil)
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		ClickID:    "clk-missing",
		Payload:    []byte(`{"goal_name":"lead"}`),
	}
	applier.ApplyBatch(context.Background(), []*domain.Event{evt})
	if evt.FraudReason != "" {
		t.Fatalf("reason %q", evt.FraudReason)
	}
	if !domain.ConversionValidationPending(evt.Payload) {
		t.Fatalf("payload %s", evt.Payload)
	}
	if !strings.Contains(string(evt.Payload), `"revenue_micro":0`) {
		t.Fatalf("payload revenue not zero: %s", evt.Payload)
	}
	if !strings.Contains(string(evt.Payload), `"payout_micro":0`) {
		t.Fatalf("payload payout not zero: %s", evt.Payload)
	}
}

func TestConversionReject_LoadClicksError_failOpen(t *testing.T) {
	campID := uuid.New()
	clicks := &stubConversionClickStore{loadErr: context.DeadlineExceeded}
	applier := NewConversionRejectApplier(conversionRejectCfg(), clicks, nil, nil)
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		ClickID:    "clk-1",
		Payload:    []byte(`{"goal_name":"lead"}`),
	}
	applier.ApplyBatch(context.Background(), []*domain.Event{evt})
	if evt.FraudReason != "" {
		t.Fatalf("reason %q", evt.FraudReason)
	}
	if !domain.ConversionValidationPending(evt.Payload) {
		t.Fatalf("payload %s", evt.Payload)
	}
	if !strings.Contains(string(evt.Payload), `"revenue_micro":0`) {
		t.Fatalf("payload revenue not zero: %s", evt.Payload)
	}
	if !strings.Contains(string(evt.Payload), `"payout_micro":0`) {
		t.Fatalf("payload payout not zero: %s", evt.Payload)
	}
}

func TestConversionReject_DatacenterIP_degradedStore(t *testing.T) {
	campID := uuid.New()
	dc := stubConversionDCChecker{datacenter: map[string]bool{"54.230.17.9": true}}
	cfg := conversionRejectCfg()
	cfg.RejectDatacenterIP = true
	applier := NewConversionRejectApplier(cfg, nil, nil, dc)
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		ClickID:    "clk-1",
		IP:         "54.230.17.9",
		Payload:    []byte(`{"goal_name":"lead"}`),
	}
	applier.ApplyBatch(context.Background(), []*domain.Event{evt})
	if evt.FraudReason != ConversionRejectDatacenterIP {
		t.Fatalf("reason %q", evt.FraudReason)
	}
}

func TestConversionReject_DatacenterIP(t *testing.T) {
	campID := uuid.New()
	clickAt := time.Now().Add(-10 * time.Second)
	clicks := &stubConversionClickStore{
		clicks: map[string]clickSnapshot{
			"clk-1": {clickID: "clk-1", campaignID: campID, createdAt: clickAt, country: "US"},
		},
	}
	dc := stubConversionDCChecker{datacenter: map[string]bool{"54.230.17.9": true}}
	cfg := conversionRejectCfg()
	cfg.RejectDatacenterIP = true
	applier := NewConversionRejectApplier(cfg, clicks, nil, dc)
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		ClickID:    "clk-1",
		IP:         "54.230.17.9",
		Payload:    []byte(`{"goal_name":"lead"}`),
	}
	applier.ApplyBatch(context.Background(), []*domain.Event{evt})
	if evt.FraudReason != ConversionRejectDatacenterIP {
		t.Fatalf("reason %q", evt.FraudReason)
	}
}

type stubConversionDCChecker struct {
	datacenter map[string]bool
}

func (s stubConversionDCChecker) IsDatacenterIP(ip string) bool {
	return s.datacenter != nil && s.datacenter[ip]
}

func TestConversionReject_NoClick(t *testing.T) {
	campID := uuid.New()
	applier := NewConversionRejectApplier(conversionRejectCfg(), &stubConversionClickStore{}, nil, nil)
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		Payload:    []byte(`{"goal_name":"lead"}`),
	}
	applier.ApplyBatch(context.Background(), []*domain.Event{evt})
	if evt.FraudReason != ConversionRejectNoClick {
		t.Fatalf("reason %q", evt.FraudReason)
	}
}

func TestConversionReject_LowTTC(t *testing.T) {
	campID := uuid.New()
	clickAt := time.Now().Add(-1 * time.Second)
	clicks := &stubConversionClickStore{
		clicks: map[string]clickSnapshot{
			"clk-1": {clickID: "clk-1", campaignID: campID, createdAt: clickAt, country: "US"},
		},
	}
	applier := NewConversionRejectApplier(conversionRejectCfg(), clicks, nil, nil)
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		ClickID:    "clk-1",
		CreatedAt:  time.Now(),
		GeoCountry: "US",
		Payload:    []byte(`{"goal_name":"lead"}`),
	}
	applier.ApplyBatch(context.Background(), []*domain.Event{evt})
	if evt.FraudReason != ConversionRejectLowTTC {
		t.Fatalf("reason %q", evt.FraudReason)
	}
	if string(evt.Payload) == "" || string(evt.Payload) == `{"goal_name":"lead"}` {
		t.Fatalf("payload not zeroed: %s", evt.Payload)
	}
}

func TestConversionReject_DuplicateGoal(t *testing.T) {
	campID := uuid.New()
	goalKey := conversionGoalKey{campaignID: campID, clickID: "clk-1", goalName: "lead"}
	clicks := &stubConversionClickStore{
		clicks: map[string]clickSnapshot{
			"clk-1": {clickID: "clk-1", campaignID: campID, createdAt: time.Now().Add(-10 * time.Second), country: "US"},
		},
		goals: map[conversionGoalKey]struct{}{
			goalKey: {},
		},
	}
	applier := NewConversionRejectApplier(conversionRejectCfg(), clicks, nil, nil)
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		ClickID:    "clk-1",
		CreatedAt:  time.Now(),
		GeoCountry: "US",
		Payload:    []byte(`{"goal_name":"lead"}`),
	}
	applier.ApplyBatch(context.Background(), []*domain.Event{evt})
	if evt.FraudReason != ConversionRejectDuplicate {
		t.Fatalf("reason %q", evt.FraudReason)
	}
}

func TestConversionReject_IPDrift(t *testing.T) {
	campID := uuid.New()
	clicks := &stubConversionClickStore{
		clicks: map[string]clickSnapshot{
			"clk-1": {clickID: "clk-1", campaignID: campID, createdAt: time.Now().Add(-10 * time.Second), country: "US"},
		},
	}
	applier := NewConversionRejectApplier(conversionRejectCfg(), clicks, nil, nil)
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		ClickID:    "clk-1",
		CreatedAt:  time.Now(),
		GeoCountry: "DE",
		Payload:    []byte(`{"goal_name":"lead"}`),
	}
	applier.ApplyBatch(context.Background(), []*domain.Event{evt})
	if evt.FraudReason != ConversionRejectIPDrift {
		t.Fatalf("reason %q", evt.FraudReason)
	}
}

func TestConversionReject_SilentRejectCampaignFlag(t *testing.T) {
	campID := uuid.New()
	applier := NewConversionRejectApplier(conversionRejectCfg(), &stubConversionClickStore{}, &stubConversionCampaignReader{
		silent: map[uuid.UUID]bool{campID: true},
	}, nil)
	evt := &domain.Event{Type: "conversion", CampaignID: campID, Payload: []byte(`{"goal_name":"lead"}`)}
	applier.ApplyBatch(context.Background(), []*domain.Event{evt})
	if !evt.SilentRejectEvent {
		t.Fatal("expected silent reject flag")
	}
}

func TestConversionReject_rejectSkipsOutboxEnqueue(t *testing.T) {
	campID := uuid.New()
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		ClickID:    "clk-1",
		Payload:    []byte(`{"goal_name":"lead"}`),
	}
	applier := NewConversionRejectApplier(conversionRejectCfg(), &stubConversionClickStore{}, nil, nil)
	applier.ApplyBatch(context.Background(), []*domain.Event{evt})

	store := &benchPostbackQuerier{}
	enq := NewConversionPostbackEnqueuer(store)
	enq.OnBatchStored(context.Background(), []*domain.Event{evt})
	if store.outboxCalls != 0 {
		t.Fatalf("outbox calls %d", store.outboxCalls)
	}
}
