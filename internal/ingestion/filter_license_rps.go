package ingestion

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/metrics"
)

// LicenseRPSFilter enforces deployment-wide JWT max_rps on the hot path.
type LicenseRPSFilter struct {
	registry licenseStateReader
}

func NewLicenseRPSFilter(registry licenseStateReader) *LicenseRPSFilter {
	return &LicenseRPSFilter{registry: registry}
}

type deploymentRPSLimiter struct {
	epoch atomic.Uint64
	count atomic.Uint64
}

var globalDeploymentRPS deploymentRPSLimiter

func (l *deploymentRPSLimiter) allow(maxRPS uint64) bool {
	if maxRPS == 0 {
		return true
	}
	now := uint64(time.Now().Unix())
	prev := l.epoch.Load()
	if prev != now {
		if l.epoch.CompareAndSwap(prev, now) {
			l.count.Store(0)
		}
	}
	n := l.count.Add(1)
	return n <= maxRPS
}

func (f *LicenseRPSFilter) Check(_ context.Context, _ *domain.Event) error {
	if f == nil || f.registry == nil {
		return nil
	}
	_, ent := f.registry.GetLicenseState()
	max := ent.Limits.MaxRPS
	if max == 0 {
		return nil
	}
	if !globalDeploymentRPS.allow(max) {
		metrics.LicenseRPSExceededTotal.Inc()
		return ErrRateLimitExceeded
	}
	return nil
}
