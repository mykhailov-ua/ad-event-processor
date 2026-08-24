package loadreport

import (
	"os"
	"strings"
)

func bpfGateLabProfile() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("BPF_GATE_PROFILE"))) {
	case "lab", "constrained", "load-test", "load_test":
		return true
	default:
		return false
	}
}

func labSkipCheck(name, value, detail string) BPFGateCheck {
	return BPFGateCheck{
		Name:   name,
		Value:  value,
		Limit:  "n/a",
		OK:     true,
		Detail: "skipped (lab profile: " + detail + ")",
	}
}

func labAwareMissingCheck(name, limit, detail string) BPFGateCheck {
	if bpfGateLabProfile() {
		return labSkipCheck(name, "missing", detail)
	}
	return strictMissingCheck(name, limit, detail)
}
