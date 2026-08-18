package loadreport

func checkBPFFDLeak(summary *bpfSummary) []BPFGateCheck {
	var checks []BPFGateCheck
	for _, s := range summary.PIDStats {
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
	for _, hw := range summary.HardwarePerf {
		if hw.Role == "tracker" {
			total += hw.CacheMisses
		}
	}
	return total
}
