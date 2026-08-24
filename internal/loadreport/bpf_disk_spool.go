package loadreport

import "strconv"

func checkBPFDiskSpoolPrometheus(prom *promClient) []BPFGateCheck {
	var checks []BPFGateCheck

	chSpoolSegments := prom.scalarOrZero("ad_ch_spool_segments")
	segVal, _ := strconv.ParseFloat(chSpoolSegments, 64)
	checks = append(checks, BPFGateCheck{
		Name:   "ch_spool_segments",
		Value:  chSpoolSegments,
		Limit:  "0",
		OK:     segVal <= 0,
		Detail: "ClickHouse spool segment backlog after settle (mmap WAL leak)",
	})

	diskGateDegraded := prom.scalarOrZero("ad_disk_gate_degraded")
	degradedVal, _ := strconv.ParseFloat(diskGateDegraded, 64)
	checks = append(checks, BPFGateCheck{
		Name:   "disk_gate_degraded",
		Value:  diskGateDegraded,
		Limit:  "0",
		OK:     degradedVal <= 0,
		Detail: "disk gate shedding TierLow appends",
	})

	brokerDiskWritable := prom.scalar(`ad_broker_disk_writable{job="broker"}`)
	if brokerDiskWritable != "na" && brokerDiskWritable != "" {
		writableVal, _ := strconv.ParseFloat(brokerDiskWritable, 64)
		checks = append(checks, BPFGateCheck{
			Name:   "broker_disk_writable",
			Value:  brokerDiskWritable,
			Limit:  "1",
			OK:     writableVal >= 1,
			Detail: "broker WAL disk must be writable",
		})
	}

	return checks
}
