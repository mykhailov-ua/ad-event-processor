package openrtb

import (
	"sync"
	"unsafe"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingest/parser"
)

const openRTBScratchMagic = 0x4f525442335f01

type ScratchSlot struct {
	Magic  uintptr
	Parsed OpenRTB3Parsed
}

var openRTBScratchPool = sync.Pool{
	New: func() any {
		return &ScratchSlot{Magic: openRTBScratchMagic}
	},
}

func AcquireScratchSlot() *ScratchSlot {
	slot := openRTBScratchPool.Get().(*ScratchSlot)
	slot.Magic = openRTBScratchMagic
	return slot
}

func ParseOpenRTB3FSMInto(out *OpenRTB3Parsed, payload []byte) bool {
	n := len(payload)
	if n < 2 {
		*out = OpenRTB3Parsed{}
		return false
	}
	_ = payload[n-1]

	out.MinBid = 0
	out.DeviceType = 1
	out.CategoryMask = 1
	out.DealIDOff = 0
	out.DealIDLen = 0
	out.ItemIDOff = 0
	out.ItemIDLen = 0
	out.RequestIDOff = 0
	out.RequestIDLen = 0
	out.TagIDOff = 0
	out.TagIDLen = 0
	out.IsOpenRTB = false
	out.OK = false

	bud := parser.NewScanBudget()
	i, ok := parser.SkipWSBudget(payload, 0, n, &bud)
	if !ok || i >= n || payload[i] != '{' {
		return false
	}

	var stack [ortbMaxDepth]ortbFrame
	depth := 0
	stack[0] = ortbFrame{parent: ortbKeyUnknown, itemIdx: -1}

	i, ok = parseOrtbObject(payload, i, n, out, &stack, &depth, &bud)
	_ = i
	if !ok && !out.IsOpenRTB {
		*out = OpenRTB3Parsed{}
		return false
	}
	if out.IsOpenRTB {
		out.OK = true
	}
	return out.OK
}

func AttachScratch(evt *domain.Event, slot *ScratchSlot) {
	if evt == nil || slot == nil {
		return
	}
	slot.Magic = openRTBScratchMagic
	evt.Scratch = unsafe.Pointer(slot)
}

func ParsedFromScratch(evt *domain.Event) (*OpenRTB3Parsed, bool) {
	if evt == nil || evt.Scratch == nil {
		return nil, false
	}
	slot := (*ScratchSlot)(evt.Scratch)
	if slot.Magic != openRTBScratchMagic {
		return nil, false
	}
	if !slot.Parsed.OK || !slot.Parsed.IsOpenRTB {
		return nil, false
	}
	return &slot.Parsed, true
}

func ReleaseScratchFromEvent(evt *domain.Event) {
	if evt == nil || evt.Scratch == nil {
		return
	}
	slot := (*ScratchSlot)(evt.Scratch)
	if slot.Magic != openRTBScratchMagic {
		return
	}
	slot.Parsed = OpenRTB3Parsed{}
	slot.Magic = openRTBScratchMagic
	openRTBScratchPool.Put(slot)
	evt.Scratch = nil
}

func ReleaseScratchSlot(slot *ScratchSlot) {
	if slot == nil {
		return
	}
	slot.Parsed = OpenRTB3Parsed{}
	slot.Magic = openRTBScratchMagic
	openRTBScratchPool.Put(slot)
}
