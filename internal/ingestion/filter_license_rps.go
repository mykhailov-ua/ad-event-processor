package ingestion

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/licensing"
	"github.com/bidshard/ad-event-processor/internal/metrics"
)

const (
	licenseRPSBurstPercent   = 10
	licenseRPSBurstWindowSec = 45
)

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

func (l *deploymentRPSLimiter) ensureBurstPool(burstCap uint64) {
	if burstCap == 0 || l.burstInit.Load() != 0 {
		return
	}
	if l.burstInit.CompareAndSwap(0, 1) {
		l.burstRemain.Store(burstCap)
	}
}

func (l *deploymentRPSLimiter) refillBurst(maxRPS, burstCap, lastCount uint64) {
	if burstCap == 0 || lastCount > maxRPS {
		return
	}
	add := maxRPS * licenseRPSBurstPercent / 100
	if add == 0 {
		add = 1
	}
	for {
		cur := l.burstRemain.Load()
		next := cur + add
		if next > burstCap {
			next = burstCap
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
	burstCap := licenseRPSBurstCap(maxRPS)
	soft := licenseRPSSoftCeil(maxRPS)
	l.ensureBurstPool(burstCap)

	now := uint64(time.Now().Unix())
	prev := l.epoch.Load()
	if prev != now {
		if l.epoch.CompareAndSwap(prev, now) {
			lastCount := l.count.Load()
			l.count.Store(0)
			l.refillBurst(maxRPS, burstCap, lastCount)
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
	maxRPS := ent.Limits.MaxRPS
	if maxRPS == 0 {
		return nil
	}
	if !licensing.SeedGateRPS(maxRPS) {
		metrics.LicenseRPSExceededTotal.Inc()
		return ErrRateLimitExceeded
	}
	if !globalDeploymentRPS.allow(maxRPS) {
		metrics.LicenseRPSExceededTotal.Inc()
		return ErrRateLimitExceeded
	}
	return nil
}
