package ingestion

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

func TestProcessTrack_accepted(t *testing.T) {
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.CampaignID = uuid.New()
	evt.Type = "click"

	out := processTrack(context.Background(), newTrackProcessor(nil, &mockRegistry{}, nil), evt, nil)
	if out.Status != trackStatusAccepted {
		t.Fatalf("status=%d want accepted", out.Status)
	}
}

func TestProcessTrack_rejected(t *testing.T) {
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.CampaignID = uuid.New()

	out := processTrack(context.Background(), newTrackProcessor(NewFilterEngine(0, &errFilter{err: ErrCampaignNotFound}), &mockRegistry{}, nil), evt, nil)
	if out.Status != trackStatusRejected || out.RejectKind != filterRejectCampaignNotFound {
		t.Fatalf("outcome=%+v", out)
	}
}

func TestProcessTrack_fraudAccepted(t *testing.T) {
	configureMockRegistryCampaign(func(c *domain.Campaign) {
		c.SilentRejectEnabled = true
	})
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.CampaignID = uuid.New()

	out := processTrack(context.Background(), newTrackProcessor(NewFilterEngine(0, &errFilter{err: ErrFraudDetected}), &mockRegistry{}, nil), evt, nil)
	if out.Status != trackStatusFraudAccepted || out.RejectKind != filterRejectFraud {
		t.Fatalf("outcome=%+v", out)
	}
	if !evt.SilentRejectEvent {
		t.Fatal("expected silent_reject_event on silent reject fraud accept")
	}
}

func TestProcessTrack_fraudHardReject_holdoutSilentRejectFlag(t *testing.T) {
	configureMockRegistryCampaign(func(c *domain.Campaign) {
		c.SilentRejectEnabled = false
	})
	t.Cleanup(resetStaticCampaignBaseline)
	evt := domain.EventPool.Get().(*domain.Event)
	evt.Reset()
	defer domain.EventPool.Put(evt)
	evt.CampaignID = uuid.New()

	out := processTrack(context.Background(), newTrackProcessor(NewFilterEngine(0, &errFilter{err: ErrFraudDetected}), &mockRegistry{}, nil), evt, nil)
	if out.Status != trackStatusRejected || out.RejectKind != filterRejectFraudBlocked {
		t.Fatalf("outcome=%+v", out)
	}
	if filterRejectSpecs[out.RejectKind].status != http.StatusForbidden {
		t.Fatal("expected 403 spec")
	}
	if evt.SilentRejectEvent {
		t.Fatal("expected silent_reject_event unset on hard fraud reject")
	}
}

func TestProcessTrack_shadowAccepted(t *testing.T) {
	geo := &MockGeoProvider{}
	fraud := NewFraudFilter(geo)
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.CampaignID = uuid.New()
	evt.IP = "1.1.1.66"
	evt.StringBuffer = make([]byte, 0, 64)

	engine := NewFilterEngine(0, fraud)
	engine.SetRegistry(&mockRegistry{})
	out := processTrack(context.Background(), newTrackProcessor(engine, &mockRegistry{}, nil), evt, nil)
	if out.Status != trackStatusAccepted {
		t.Fatalf("status=%d want accepted shadow", out.Status)
	}
	if !evt.ShadowEvent {
		t.Fatal("expected shadow flag")
	}
}

func TestProcessTrack_internalError(t *testing.T) {
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.CampaignID = uuid.New()

	out := processTrack(context.Background(), newTrackProcessor(NewFilterEngine(0, &errFilter{err: errors.New("unexpected")}), &mockRegistry{}, nil), evt, nil)
	if out.Status != trackStatusInternalError {
		t.Fatalf("status=%d", out.Status)
	}
}

func TestProcessTrack_infraReject(t *testing.T) {
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.CampaignID = uuid.New()

	out := processTrack(context.Background(), newTrackProcessor(NewFilterEngine(0, &errFilter{err: database.ErrRedisCircuitOpen}), &mockRegistry{}, nil), evt, nil)
	if out.Status != trackStatusRejected || out.RejectKind != filterRejectInfra {
		t.Fatalf("outcome=%+v", out)
	}
	if filterRejectSpecs[out.RejectKind].status != http.StatusServiceUnavailable {
		t.Fatal("expected 503 spec")
	}
}

func TestProcessTrack_filterTimeout(t *testing.T) {
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.CampaignID = uuid.New()

	out := processTrack(context.Background(), newTrackProcessor(NewFilterEngine(50*time.Millisecond, &slowFilter{delay: 200 * time.Millisecond}), &mockRegistry{}, nil), evt, nil)
	if out.Status != trackStatusRejected || out.RejectKind != filterRejectTimeout {
		t.Fatalf("outcome=%+v", out)
	}
	if filterRejectSpecs[out.RejectKind].status != http.StatusGatewayTimeout {
		t.Fatal("expected 504 spec")
	}
}
