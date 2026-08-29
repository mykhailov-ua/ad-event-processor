package ingest

import (
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestClassifyFilterErr_HandlerMatrix(t *testing.T) {
	cfg := &config.Config{
		MaxRequestBodySize: 1024 * 1024,
		FilterTimeoutMs:    50,
		StreamMaxLen:       1000,
	}
	sharder := NewStaticSlotSharder(4)

	for _, tc := range rejectMatrixCases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			timeout := time.Duration(cfg.FilterTimeoutMs) * time.Millisecond
			if tc.name == "filter_timeout" {
				timeout = 50 * time.Millisecond
			}
			var filter EventFilter
			switch tc.name {
			case "filter_timeout":
				filter = &slowFilter{delay: 200 * time.Millisecond}
			case "redis_circuit":
				filter = &errFilter{err: database.ErrRedisCircuitOpen}
			default:
				filter = tc.filter
			}
			registry := domain.CampaignRegistry(&mockRegistry{})
			body := []byte(`{"campaign_id":"` + uuid.NewString() + `","type":"click","click_id":"c1"}`)
			if tc.fraudCase {
				campID := uuid.New()
				body = []byte(`{"campaign_id":"` + campID.String() + `","type":"click","click_id":"c1"}`)
				registry = stubCampaignRegistry{
					ok: true,
					camp: &domain.Campaign{
						ID:                  campID,
						SilentRejectEnabled: tc.silentReject,
					},
				}
			}
			h := NewAdsPacketHandler(cfg, registry, NewFilterEngine(timeout, filter), nil, nil, sharder, "fraud-stream", nil)
			status, _ := PostTrackGnetJSON(h, body)
			assert.Equal(t, tc.wantStatus, status)
		})
	}
}
