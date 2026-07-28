//go:build espx_bpf_trace

package traceprobe

// USDT-style markers for dev BPF uprobes. Build tracker with -tags espx_bpf_trace.
// Symbols must stay //go:noinline so uprobes can attach.

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
	// Prevent the compiler from eliding parameters before uprobe reads registers.
	if markerID == ^uint32(0) && slot == ^uint32(0) {
		println()
	}
}
