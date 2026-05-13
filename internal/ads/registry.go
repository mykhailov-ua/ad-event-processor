package ads

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mykhailov-ua/ad-event-processor/internal/ads/repository"
)

// Registry maintains an in-memory map of active campaigns for high-performance lookups.
// Chosen to eliminate database round-trips for campaign validation in the hot path.
type campaignInfo struct {
	customerID uuid.UUID
	status     repository.CampaignStatusType
}

type Registry struct {
	repo          repository.Querier
	data          map[uuid.UUID]campaignInfo
	manuallyAdded map[uuid.UUID]bool // Tracks IDs added via Add() that haven't been seen in DB yet
	mu            sync.RWMutex
	wg            sync.WaitGroup
}

// NewRegistry initializes the registry with optimized initial capacities.
func NewRegistry(repo repository.Querier) *Registry {
	return &Registry{
		data:          make(map[uuid.UUID]campaignInfo, 100_000),
		manuallyAdded: make(map[uuid.UUID]bool),
		repo:          repo,
	}
}

// Exists checks if a campaign is registered and currently active.
func (r *Registry) Exists(id uuid.UUID) bool {
	r.mu.RLock()
	info, ok := r.data[id]
	r.mu.RUnlock()
	return ok && info.status == repository.CampaignStatusTypeACTIVE
}

// GetCustomerID retrieves the customer ID associated with a specific campaign.
func (r *Registry) GetCustomerID(campaignID uuid.UUID) (uuid.UUID, bool) {
	r.mu.RLock()
	info, ok := r.data[campaignID]
	r.mu.RUnlock()
	if !ok {
		return uuid.Nil, false
	}
	return info.customerID, true
}

// Add manually inserts a campaign into the registry and marks it as manually added.
// This ensures the campaign persists through background syncs until it is confirmed by the database.
func (r *Registry) Add(id, customerID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	info := campaignInfo{customerID: customerID, status: repository.CampaignStatusTypeACTIVE}
	r.data[id] = info
	r.manuallyAdded[id] = true
}

// Sync fetches all active campaigns from the database and atomicaly merges them with manually added entries.
// Chosen to maintain the database as the source of truth while supporting immediate consistency for API-driven updates.
func (r *Registry) Sync(ctx context.Context) (int, error) {
	rows, err := r.repo.ListActiveCampaigns(ctx)
	if err != nil {
		return 0, err
	}

	fresh := make(map[uuid.UUID]campaignInfo, len(rows))
	for _, row := range rows {
		id := uuid.UUID(row.ID.Bytes)
		fresh[id] = campaignInfo{
			customerID: uuid.UUID(row.CustomerID.Bytes),
			status:     row.Status,
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// 1. Remove confirmed items from manuallyAdded set
	for id := range fresh {
		delete(r.manuallyAdded, id)
	}

	// 2. Merge remaining manual additions into the fresh map
	for id := range r.manuallyAdded {
		if info, ok := r.data[id]; ok {
			fresh[id] = info
		}
	}

	r.data = fresh
	return len(fresh), nil
}

// StartSync initiates a background goroutine to periodically synchronize with the database.
func (r *Registry) StartSync(ctx context.Context, interval time.Duration) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				count, err := r.Sync(ctx)
				if err != nil {
					slog.Error("campaign registry sync failed", "error", err)
					continue
				}
				slog.Debug("campaign registry synced", "campaigns", count)
			}
		}
	}()
}

// Wait blocks until the synchronization goroutine has exited gracefully.
func (r *Registry) Wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
