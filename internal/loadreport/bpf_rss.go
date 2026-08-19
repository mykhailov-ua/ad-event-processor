package loadreport

import "strconv"

func checkBPFRSSChecks(summary *bpfSummary, roles []string, maxRSSDeltaKB int64) []BPFGateCheck {
	var checks []BPFGateCheck
	want := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		want[r] = struct{}{}
	}

	for i := range summary.ProcSamples {
		s := &summary.ProcSamples[i]
		if _, ok := want[s.Role]; !ok {
			continue
		}
		if s.RSSDelta > maxRSSDeltaKB {
			checks = append(checks, BPFGateCheck{
				Name:   s.Role + "_rss_delta_kb",
				Value:  strconv.FormatInt(s.RSSDelta, 10),
				Limit:  strconv.FormatInt(maxRSSDeltaKB, 10),
				OK:     false,
				Detail: "RSS grew beyond session budget (memory leak or heap growth)",
			})
		}
		if s.MajFlt > 0 {
			checks = append(checks, BPFGateCheck{
				Name:   s.Role + "_major_page_faults",
				Value:  strconv.FormatInt(s.MajFlt, 10),
				Limit:  "0",
				OK:     false,
				Detail: "major page faults observed (memory pressure or swap)",
			})
		}
	}
	return checks
}
