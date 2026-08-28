package ingestion

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type traceFilter struct {
	name  string
	trace *[]string
	fail  error
}

func (f *traceFilter) Check(ctx context.Context, evt *domain.Event) error {
	*f.trace = append(*f.trace, f.name)
	return f.fail
}

func TestFilterEngine_ProductionOrder(t *testing.T) {
	var order []string
	emergency := &traceFilter{name: "emergency", trace: &order}
	geo := &traceFilter{name: "geo", trace: &order}
	schedule := &traceFilter{name: "schedule", trace: &order}
	unified := &traceFilter{name: "unified", trace: &order, fail: ErrRateLimitExceeded}

	engine := NewFilterEngine(time.Second, emergency, geo, schedule, unified)
	evt := &domain.Event{CampaignID: uuid.New()}

	err := engine.Check(context.Background(), evt)
	require.ErrorIs(t, err, ErrRateLimitExceeded)
	assert.Equal(t, []string{"emergency", "geo", "schedule", "unified"}, order)
}

func TestFilterEngine_TrackerSegmentAfterLocalFilters(t *testing.T) {
	var order []string
	engine := NewFilterEngine(time.Second,
		&traceFilter{name: "license", trace: &order},
		&traceFilter{name: "license_rps", trace: &order},
		&traceFilter{name: "emergency", trace: &order},
		&traceFilter{name: "geo", trace: &order},
		&traceFilter{name: "schedule", trace: &order},
		&traceFilter{name: "vpp", trace: &order},
		&traceFilter{name: "fraud", trace: &order},
		&traceFilter{name: "residential", trace: &order},
		&traceFilter{name: "tcp_mss", trace: &order},
		&traceFilter{name: "device", trace: &order},
		&traceFilter{name: "l7_wire", trace: &order},
		&traceFilter{name: "consent", trace: &order},
		&traceFilter{name: "segment", trace: &order},
		&traceFilter{name: "entitlements", trace: &order},
		&traceFilter{name: "unified", trace: &order},
	)
	evt := &domain.Event{CampaignID: uuid.New()}
	require.NoError(t, engine.Check(context.Background(), evt))
	assert.Equal(t, []string{
		"license", "license_rps", "emergency", "geo", "schedule", "vpp", "fraud",
		"residential", "tcp_mss", "device", "l7_wire", "consent", "segment", "entitlements", "unified",
	}, order)
}

func TestFilterEngine_segmentSkippedWhenGeoRejects(t *testing.T) {
	var order []string
	engine := NewFilterEngine(time.Second,
		&traceFilter{name: "geo", trace: &order, fail: ErrGeoBlocked},
		&traceFilter{name: "schedule", trace: &order},
		&traceFilter{name: "vpp", trace: &order},
		&traceFilter{name: "fraud", trace: &order},
		&traceFilter{name: "device", trace: &order},
		&traceFilter{name: "consent", trace: &order},
		&traceFilter{name: "segment", trace: &order},
		&traceFilter{name: "entitlements", trace: &order},
		&traceFilter{name: "unified", trace: &order},
	)
	evt := &domain.Event{CampaignID: uuid.New()}
	err := engine.Check(context.Background(), evt)
	require.ErrorIs(t, err, ErrGeoBlocked)
	assert.Equal(t, []string{"geo"}, order)
}

func TestFilterEngine_DeadlineShortCircuit(t *testing.T) {
	var order []string
	engine := NewFilterEngine(5*time.Millisecond,
		&traceFilter{name: "fast", trace: &order},
		&slowFilter{delay: 20 * time.Millisecond},
		&traceFilter{name: "never", trace: &order, fail: ErrBudgetExhausted},
	)
	evt := &domain.Event{CampaignID: uuid.New()}
	err := engine.Check(context.Background(), evt)
	require.ErrorIs(t, err, ErrFilterTimeout)
	assert.Equal(t, []string{"fast"}, order)
}
