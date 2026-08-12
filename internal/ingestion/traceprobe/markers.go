//go:build ad_event_processor_bpf_trace

package traceprobe

//go:noinline
func ProcessTrackEnter(slot uint32) {
	markEnter(1, slot)
}

//go:noinline
func ProcessTrackExit(slot uint32) {
	markExit(2, slot)
}

//go:noinline
func FilterCheckEnter(slot uint32) {
	markEnter(3, slot)
}

//go:noinline
func FilterCheckExit(slot uint32) {
	markExit(4, slot)
}

//go:noinline
func markEnter(markerID, slot uint32) {
	mark(markerID, slot)
}

//go:noinline
func markExit(markerID, slot uint32) {
	mark(markerID, slot)
}

//go:noinline
func mark(markerID, slot uint32) {
	if markerID == ^uint32(0) && slot == ^uint32(0) {
		println()
	}
}
