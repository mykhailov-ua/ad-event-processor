package edge

import (
	"encoding/binary"
	"errors"
	"os"
	"time"

	"github.com/cilium/ebpf/ringbuf"
)

type ViolationHandler struct {
	onEvent func(ViolationEvent) error
}

func NewViolationHandler(onEvent func(ViolationEvent) error) *ViolationHandler {
	return &ViolationHandler{onEvent: onEvent}
}

// Drain reads violations ringbuf until idle window; dedupes by host per drain pass
// before onEvent (typically RecordAutoBan -> Redis blacklist:auto).
func (h *ViolationHandler) Drain(rd *ringbuf.Reader, idle time.Duration) (int, error) {
	if rd == nil || h.onEvent == nil {
		return 0, nil
	}
	seen := make(map[string]struct{})
	deadline := time.Now().Add(idle)
	var handled int

	for time.Now().Before(deadline) {
		rd.SetDeadline(time.Now().Add(1 * time.Millisecond))
		record, err := rd.Read()
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue
			}
			if isRingbufClosed(err) {
				return handled, nil
			}
			return handled, err
		}
		evt, ok := decodeViolation(record.RawSample)
		if !ok {
			continue
		}
		host := ViolationHost(evt)
		if host == "" {
			continue
		}
		if _, dup := seen[host]; dup {
			continue
		}
		seen[host] = struct{}{}
		if err := h.onEvent(evt); err != nil {
			return handled, err
		}
		handled++
		deadline = time.Now().Add(idle)
	}
	return handled, nil
}

func decodeViolation(raw []byte) (ViolationEvent, bool) {
	if len(raw) < ViolationEventWireSize {
		return ViolationEvent{}, false
	}
	var evt ViolationEvent
	evt.TSNs = binary.LittleEndian.Uint64(raw[0:8])
	evt.Family = raw[8]
	evt.Reason = raw[9]
	copy(evt.Addr[:], raw[12:28])
	return evt, true
}

func isRingbufClosed(err error) bool {
	return errors.Is(err, ringbuf.ErrClosed)
}
