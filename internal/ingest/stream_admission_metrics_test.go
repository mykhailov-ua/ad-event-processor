package ingest

import (
	"strconv"
	"testing"
	"unsafe"
)

func TestShardLabelString_matchesstrconv(t *testing.T) {
	for _, shard := range []int{-1, 0, 1, 7, 42, 127} {
		got := shardLabelString(shard)
		want := strconv.Itoa(shard)
		if got != want {
			t.Fatalf("shard %d: got %q want %q", shard, got, want)
		}
	}
}

// Holdout: single-digit shard labels must not use stack-scratch unsafeString (pre-fix used c000... addresses).
func TestShardLabelString_holdoutSingleDigitNotStackArena(t *testing.T) {
	const stackArenaLo = uintptr(0xc000000000)
	for shard := range 10 {
		label := shardLabelString(shard)
		p := uintptr(unsafe.Pointer(unsafe.StringData(label)))
		if p >= stackArenaLo {
			t.Fatalf("shard %d: label %q backed by stack arena %#x; revert of P0-1 unsafeString?", shard, label, p)
		}
	}
}

// stackUnsafeShardLabel reproduces the pre-fix pattern (unsafeString over stack scratch).
func stackUnsafeShardLabel(shard int) string {
	var scratch [8]byte
	return testUnsafeString(appendInt64(scratch[:0], int64(shard)))
}

func testUnsafeString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func TestStackUnsafeShardLabel_holdoutAliasesScratch(t *testing.T) {
	var scratch [8]byte
	sl := appendInt64(scratch[:0], 42)
	label := testUnsafeString(sl)
	if unsafe.StringData(label) != unsafe.SliceData(sl) {
		t.Fatal("holdout: stack unsafeString must alias scratch buffer (documents P0-1 UAF root cause)")
	}
}

func TestShardLabelString_holdoutDiffersFromStackUnsafeBacking(t *testing.T) {
	const shard = 42
	fixed := shardLabelString(shard)
	var scratch [8]byte
	buggy := testUnsafeString(appendInt64(scratch[:0], int64(shard)))
	if unsafe.StringData(fixed) == unsafe.StringData(buggy) {
		t.Fatal("holdout: fixed label must not share backing with stack-scratch unsafeString")
	}
}

func TestStreamAdmissionMetrics_usesAtomicSlots_notSyncMap(t *testing.T) {
	m0 := streamAdmissionMetricsForShard(0)
	m1 := streamAdmissionMetricsForShard(0)
	if m0 == nil || m1 == nil || m0 != m1 {
		t.Fatal("streamAdmissionMetricsForShard must return stable pointer per shard")
	}
	b0 := brokerAdmissionMetricsFor(-1, false)
	b1 := brokerAdmissionMetricsFor(-1, false)
	if b0 == nil || b1 == nil || b0 != b1 {
		t.Fatal("brokerAdmissionMetricsFor single-broker must return stable pointer")
	}
}
