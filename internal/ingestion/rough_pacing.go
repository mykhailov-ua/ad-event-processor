package ingestion

import (
	"hash/maphash"
	"sync"
	"sync/atomic"
	"time"

	"espx/internal/domain"

	"github.com/google/uuid"
)

const roughPacingCells = 4096

type roughPacingCell struct {
	campaignID uuid.UUID
	dayKey     uint32
	spentMicro atomic.Int64
}

type RoughPacingGate struct {
	seed  maphash.Seed
	mu    sync.Mutex
	cells [roughPacingCells]roughPacingCell
}

func NewRoughPacingGate() *RoughPacingGate {
	return &RoughPacingGate{seed: maphash.MakeSeed()}
}

func (g *RoughPacingGate) dayKey(t time.Time) uint32 {
	y, m, d := t.Date()
	return uint32(y)*10000 + uint32(m)*100 + uint32(d)
}

func (g *RoughPacingGate) cellFor(campaignID uuid.UUID, day uint32) *roughPacingCell {
	var h maphash.Hash
	h.SetSeed(g.seed)
	h.Write(campaignID[:])
	idx := h.Sum64() % roughPacingCells
	cell := &g.cells[idx]
	g.mu.Lock()
	if cell.campaignID != campaignID || cell.dayKey != day {
		cell.campaignID = campaignID
		cell.dayKey = day
		cell.spentMicro.Store(0)
	}
	g.mu.Unlock()
	return cell
}

func (g *RoughPacingGate) Allow(campaignID uuid.UUID, amountMicro, dailyBudgetMicro int64, hour int) bool {
	if g == nil || dailyBudgetMicro <= 0 || amountMicro <= 0 {
		return true
	}
	if hour < 1 {
		hour = 1
	} else if hour > 24 {
		hour = 24
	}
	cumulativeLimit := (dailyBudgetMicro * int64(hour)) / 24
	now := CachedTimeUTC()
	cell := g.cellFor(campaignID, g.dayKey(now))
	spent := cell.spentMicro.Add(amountMicro)
	if spent > cumulativeLimit {
		cell.spentMicro.Add(-amountMicro)
		return false
	}
	return true
}

func (f *UnifiedFilter) SetRoughPacingGate(g *RoughPacingGate) {
	f.roughPacing = g
}

func (f *UnifiedFilter) checkGoRoughPacing(evt *domain.Event, camp *domain.Campaign, amountMicro int64) error {
	if f == nil || f.roughPacing == nil || camp == nil || !camp.RoughPacingEnabled() {
		return nil
	}
	if camp.PacingMode != domain.PacingModeEven {
		return nil
	}
	if evt.Type != "impression" && evt.Type != "click" {
		return nil
	}
	hr := CachedTimeUTC().Hour() + 1
	if !f.roughPacing.Allow(camp.ID, amountMicro, camp.DailyBudgetMicro, hr) {
		return ErrPacingExhausted
	}
	return nil
}
