package loadreport

const pgWirePort = 5432

func pgWireStats(network []networkStat, role string) (connects, sendtoCalls, sendtoBytes int64) {
	for _, n := range network {
		if n.Role != role || n.Dport != pgWirePort {
			continue
		}
		connects += n.Connects
		sendtoCalls += n.SendtoCalls
		sendtoBytes += n.SendtoBytes
	}
	return connects, sendtoCalls, sendtoBytes
}

func checkBPFPGWireChecks(summary *bpfSummary) []BPFGateCheck {
	var checks []BPFGateCheck
	for _, role := range []string{"processor", "control"} {
		connects, sendtoCalls, sendtoBytes := pgWireStats(summary.Network, role)
		if connects == 0 && sendtoCalls == 0 && sendtoBytes == 0 {
			continue
		}
		limit := int64(20)
		if role == "control" {
			limit = 50
		}
		checks = append(checks, BPFGateCheck{
			Name:   role + "_pg_connects",
			Value:  strconvFormatInt(connects),
			Limit:  strconvFormatInt(limit),
			OK:     connects <= limit,
			Detail: "PG wire connect() to :5432; pool reuse expected on cold path",
		})
		if sendtoCalls > 0 {
			checks = append(checks, BPFGateCheck{
				Name:   role + "_pg_sendto_calls",
				Value:  strconvFormatInt(sendtoCalls),
				Limit:  "n/a",
				OK:     true,
				Detail: "PG wire sendto() samples observed",
			})
		}
		if sendtoBytes > 0 {
			checks = append(checks, BPFGateCheck{
				Name:   role + "_pg_sendto_bytes",
				Value:  strconvFormatInt(sendtoBytes),
				Limit:  "n/a",
				OK:     true,
				Detail: "PG wire sendto() bytes observed",
			})
		}
	}
	return checks
}
