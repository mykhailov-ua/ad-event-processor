package edge

import (
	"fmt"
	"os"
	"testing"

	"github.com/cilium/ebpf/rlimit"
)

func TestMain(m *testing.M) {
	if err := rlimit.RemoveMemlock(); err != nil {
		fmt.Fprintf(os.Stderr, "edge_test: WARN RemoveMemlock: %v\n", err)
	}
	if disabled, _ := readUnprivilegedBPFDisabled(); disabled >= 2 {
		fmt.Fprintf(os.Stderr, "edge_test: WARN kernel.unprivileged_bpf_disabled=%d — BPF map tests need root/CAP_BPF (prog_test skips)\n", disabled)
	}
	os.Exit(m.Run())
}

func readUnprivilegedBPFDisabled() (int, error) {
	b, err := os.ReadFile("/proc/sys/kernel/unprivileged_bpf_disabled")
	if err != nil {
		return 0, err
	}
	var v int
	if _, err := fmt.Sscanf(string(b), "%d", &v); err != nil {
		return 0, err
	}
	return v, nil
}
