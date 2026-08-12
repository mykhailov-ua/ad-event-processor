package ingestion

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

// mockRegistry is a no-op CampaignRegistry for handler tests and chaos load harnesses.
type mockRegistry struct{}

func (m *mockRegistry) Exists(id uuid.UUID) bool { return true }
func (m *mockRegistry) Add(id, customerID uuid.UUID, brandID *uuid.UUID, brandFcapKey string, pacingMode domain.PacingMode, dailyBudget int64, timezone string, freqLimit, freqWindow int32, targetCountries []string) {
}
func (m *mockRegistry) GetCustomerID(id uuid.UUID) (uuid.UUID, bool) { return uuid.Nil, true }

var (
	staticCampaignMu sync.RWMutex
	staticCampaign   = &domain.Campaign{CustomerID: uuid.Nil, Location: time.UTC}
	cachedMockCamp   atomic.Pointer[domain.Campaign]
)

func enrichMockCampaign(cp *domain.Campaign) {
	if cp.Location == nil {
		cp.Location = time.UTC
	}
	if cp.IDStr == "" {
		cp.IDStr = cp.ID.String()
	}
	if cp.IDStrAny == nil {
		cp.IDStrAny = cp.IDStr
	}
	if cp.CustomerIDStr == "" {
		cp.CustomerIDStr = cp.CustomerID.String()
	}
	if cp.CustomerIDStrAny == nil {
		cp.CustomerIDStrAny = cp.CustomerIDStr
	}
	if cp.BudgetCampaignKey == "" {
		cp.BudgetCampaignKey = "budget:campaign:" + cp.IDStr
	}
	if cp.CampaignSyncKey == "" {
		cp.CampaignSyncKey = "budget:sync:campaign:" + cp.IDStr
	}
	if cp.CustomerSyncKey == "" {
		cp.CustomerSyncKey = "budget:sync:customer:" + cp.CustomerIDStr
	}
	if cp.FcapKeyPrefix == "" {
		if cp.BrandFcapKey != "" {
			cp.FcapKeyPrefix = cp.BrandFcapKey + ":u:"
		} else {
			cp.FcapKeyPrefix = "fcap:c:" + cp.IDStr + ":u:"
		}
	}
	if cp.DailySpendKeyPrefix == "" {
		cp.DailySpendKeyPrefix = "budget:daily_spent:campaign:" + cp.IDStr + ":"
	}
	if cp.DailyBudgetMicroAny == nil && cp.DailyBudgetMicro != 0 {
		cp.DailyBudgetMicroAny = cp.DailyBudgetMicro
	}
}

func (m *mockRegistry) GetCampaign(id uuid.UUID) (*domain.Campaign, bool) {
	if got := cachedMockCamp.Load(); got != nil && got.ID == id {
		if got.BudgetCampaignKey == "" {
			cp := *got
			enrichMockCampaign(&cp)
			cachedMockCamp.Store(&cp)
		}
		return cachedMockCamp.Load(), true
	}

	staticCampaignMu.RLock()
	defer staticCampaignMu.RUnlock()

	cp := *staticCampaign
	cp.ID = id
	enrichMockCampaign(&cp)

	cachedMockCamp.Store(&cp)
	return cachedMockCamp.Load(), true
}
func (m *mockRegistry) Sync(ctx context.Context) (int, error)                 { return 0, nil }
func (m *mockRegistry) StartSync(ctx context.Context, interval time.Duration) {}
func (m *mockRegistry) Wait(ctx context.Context) error                        { return nil }
