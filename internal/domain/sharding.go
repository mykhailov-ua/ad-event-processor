package domain

import (
	"sync/atomic"

	"github.com/google/uuid"
)

type slotTable [1024]uint8

type SlotTable = slotTable

func BuildSlotTable(numBuckets int) *SlotTable {
	return buildSlotTable(numBuckets)
}

type SlotMapSnapshot struct {
	Table        slotTable
	Version      int32
	MigrationGen int64
}

type Sharder interface {
	GetShard(id uuid.UUID) int
}

type JumpHashSharder struct {
	numBuckets int
}

type StaticSlotSharder struct {
	snapshot atomic.Value
}

func buildSlotTable(numBuckets int) *slotTable {
	if numBuckets <= 0 {
		numBuckets = 1
	}
	var t slotTable
	for i := range t {
		t[i] = uint8(i % numBuckets)
	}
	return &t
}

func (s *StaticSlotSharder) loadSnapshot() *SlotMapSnapshot {
	if v := s.snapshot.Load(); v != nil {
		return v.(*SlotMapSnapshot)
	}
	fallback := &SlotMapSnapshot{Table: *buildSlotTable(1)}
	return fallback
}

func NewStaticSlotSharder(numBuckets int) *StaticSlotSharder {
	sh := &StaticSlotSharder{}
	sh.snapshot.Store(&SlotMapSnapshot{
		Table:   *buildSlotTable(numBuckets),
		Version: 0,
	})
	return sh
}

func (s *StaticSlotSharder) GetShard(id uuid.UUID) int {
	key := crc32Castagnoli(&id)
	slot := key & 1023
	snap := s.loadSnapshot()
	return int(snap.Table[slot])
}

func (s *StaticSlotSharder) SnapshotVersion() int32 {
	return s.loadSnapshot().Version
}

func (s *StaticSlotSharder) SwapSnapshot(version int32, table *slotTable, migrationGen int64) {
	var t slotTable
	if table != nil {
		t = *table
	} else {
		t = s.loadSnapshot().Table
	}
	s.snapshot.Store(&SlotMapSnapshot{
		Table:        t,
		Version:      version,
		MigrationGen: migrationGen,
	})
}

func (s *StaticSlotSharder) ReloadFromModulo(numBuckets int) {
	s.SwapSnapshot(0, buildSlotTable(numBuckets), 0)
}

func (s *StaticSlotSharder) StoreSlotMap(table *[1024]uint16) {
	if table == nil {
		return
	}
	prev := s.loadSnapshot()
	var st slotTable
	for i, v := range table {
		st[i] = uint8(v)
	}
	s.SwapSnapshot(prev.Version, &st, prev.MigrationGen)
}

func (s *StaticSlotSharder) SetActiveVersion(version int32) {
	prev := s.loadSnapshot()
	t := prev.Table
	s.SwapSnapshot(version, &t, prev.MigrationGen)
}

func (s *StaticSlotSharder) ActiveVersion() int32 {
	return s.loadSnapshot().Version
}

func (s *StaticSlotSharder) Snapshot() SlotMapSnapshot {
	return *s.loadSnapshot()
}

func NewJumpHashSharder(numBuckets int) *JumpHashSharder {
	if numBuckets <= 0 {
		numBuckets = 1
	}
	return &JumpHashSharder{numBuckets: numBuckets}
}

func (s *JumpHashSharder) GetShard(id uuid.UUID) int {
	if s.numBuckets <= 1 {
		return 0
	}

	key := uint64(crc32Castagnoli(&id))

	return int(jumpHash(key, int32(s.numBuckets)))
}

func jumpHash(key uint64, numBuckets int32) int32 {
	var b int64 = -1
	var j int64
	for j < int64(numBuckets) {
		b = j
		key = key*2862933555777941757 + 1
		j = int64(float64(b+1) * (float64(1<<31) / float64((key>>33)+1)))
	}
	return int32(b)
}

func CRC32Castagnoli(data *uuid.UUID) uint32 {
	return crc32Castagnoli(data)
}
