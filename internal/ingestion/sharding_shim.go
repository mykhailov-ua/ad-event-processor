package ingestion

import "github.com/bidshard/ad-event-processor/internal/domain"

type (
	Sharder           = domain.Sharder
	StaticSlotSharder = domain.StaticSlotSharder
	JumpHashSharder   = domain.JumpHashSharder
	SlotMapSnapshot   = domain.SlotMapSnapshot
)

func NewStaticSlotSharder(numBuckets int) *StaticSlotSharder {
	return domain.NewStaticSlotSharder(numBuckets)
}

func NewJumpHashSharder(numBuckets int) *JumpHashSharder {
	return domain.NewJumpHashSharder(numBuckets)
}

type slotTable = domain.SlotTable

func buildSlotTable(numBuckets int) *slotTable {
	return domain.BuildSlotTable(numBuckets)
}
