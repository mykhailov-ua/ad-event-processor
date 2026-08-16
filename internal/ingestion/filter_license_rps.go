package ingestion

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/licensing"
	"github.com/bidshard/ad-event-processor/internal/metrics"
)

// LicenseRPS burst policy: short spikes above JWT max_rps without immediate 429.
// Sustained rate stays at max_rps; per-second ceiling is max_rps + 10%.
// Burst credits cover the overage for ~45s of elevated traffic, then refill on quiet seconds.
const (
	licenseRPSBurstPercent   = 10
	licenseRPSBurstWindowSec = 45
)

// LicenseRPSFilter enforces deployment-wide JWT max_rps on the hot path.
type LicenseRPSFilter struct {
	registry licenseStateReader
}

func NewLicenseRPSFilter(registry licenseStateReader) *LicenseRPSFilter {
	return &LicenseRPSFilter{registry: registry}
}

type deploymentRPSLimiter struct {
	epoch       atomic.Uint64
	count       atomic.Uint64
	burstRemain atomic.Uint64
	burstInit   atomic.Uint32
}

var globalDeploymentRPS deploymentRPSLimiter

func licenseRPSSoftCeil(maxRPS uint64) uint64 {
	if maxRPS == 0 {
		return 0
	}
	extra := maxRPS * licenseRPSBurstPercent / 100
	if extra == 0 {
		extra = 1
	}
	return maxRPS + extra
}

func licenseRPSBurstCap(maxRPS uint64) uint64 {
	if maxRPS == 0 {
		return 0
	}
	return maxRPS * licenseRPSBurstWindowSec * uint64(licenseRPSBurstPercent) / 100
}

func (l *deploymentRPSLimiter) resetForTests() {
	l.epoch.Store(uint64(time.Now().Unix()))
	l.count.Store(0)
	l.burstRemain.Store(0)
	l.burstInit.Store(0)
}

func (l *deploymentRPSLimiter) ensureBurstPool(cap uint64) {
	if cap == 0 || l.burstInit.Load() != 0 {
		return
	}
	if l.burstInit.CompareAndSwap(0, 1) {
		l.burstRemain.Store(cap)
	}
}

func (l *deploymentRPSLimiter) refillBurst(maxRPS, cap, lastCount uint64) {
	if cap == 0 || lastCount > maxRPS {
		return
	}
	add := maxRPS * licenseRPSBurstPercent / 100
	if add == 0 {
		add = 1
	}
	for {
		cur := l.burstRemain.Load()
		next := cur + add
		if next > cap {
			next = cap
		}
		if next == cur || l.burstRemain.CompareAndSwap(cur, next) {
			return
		}
	}
}

func (l *deploymentRPSLimiter) consumeBurst() bool {
	for {
		cur := l.burstRemain.Load()
		if cur == 0 {
			return false
		}
		if l.burstRemain.CompareAndSwap(cur, cur-1) {
			return true
		}
	}
}

func (l *deploymentRPSLimiter) allow(maxRPS uint64) bool {
	if maxRPS == 0 {
		return true
	}
	cap := licenseRPSBurstCap(maxRPS)
	soft := licenseRPSSoftCeil(maxRPS)
	l.ensureBurstPool(cap)

	now := uint64(time.Now().Unix())
	prev := l.epoch.Load()
	if prev != now {
		if l.epoch.CompareAndSwap(prev, now) {
			lastCount := l.count.Load()
			l.count.Store(0)
			l.refillBurst(maxRPS, cap, lastCount)
		}
	}

	n := l.count.Add(1)
	if n <= maxRPS {
		return true
	}
	if n > soft {
		return false
	}
	return l.consumeBurst()
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
	if !licensing.SeedGateRPS(max) {
		metrics.LicenseRPSExceededTotal.Inc()
		return ErrRateLimitExceeded
	}
	if !globalDeploymentRPS.allow(max) {
		metrics.LicenseRPSExceededTotal.Inc()
		return ErrRateLimitExceeded
	}
	return nil
}
