package loadreport

func checkBPFFDLeak(summary *bpfSummary) []BPFGateCheck {
	if bpfGateLabProfile() {
		return nil
	}
	var checks []BPFGateCheck
	for i := range summary.PIDStats {
		s := &summary.PIDStats[i]
		if s.Role != "tracker" && s.Role != "processor" {
			continue
		}
		if s.FDOpenPerSec <= 0 || s.FDClosePerSec <= 0 {
			continue
		}
		if s.FDOpenPerSec > s.FDClosePerSec*2 {
			checks = append(checks, BPFGateCheck{
				Name:   s.Role + "_fd_open_close_imbalance",
				Value:  formatFloat(s.FDOpenPerSec, 1),
				Limit:  formatFloat(s.FDClosePerSec*2, 1),
				OK:     false,
				Detail: "openat rate >> close rate under load (FD/socket leak)",
			})
		}
	}
	return checks
}

func trackerHWCacheMisses(summary *bpfSummary) uint64 {
	var total uint64
	for i := range summary.HardwarePerf {
		if summary.HardwarePerf[i].Role == "tracker" {
			total += summary.HardwarePerf[i].CacheMisses
		}
	}
	return total
}
