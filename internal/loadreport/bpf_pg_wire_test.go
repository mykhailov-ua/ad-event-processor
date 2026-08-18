package loadreport

import "testing"

func TestPGWireStats(t *testing.T) {
	network := []networkStat{
		{Role: "processor", Dport: 5432, Connects: 3, SendtoCalls: 10, SendtoBytes: 4096},
		{Role: "processor", Dport: 6379, Connects: 1},
		{Role: "tracker", Dport: 5432, Connects: 99},
	}
	c, calls, bytes := pgWireStats(network, "processor")
	if c != 3 || calls != 10 || bytes != 4096 {
		t.Fatalf("got connects=%d calls=%d bytes=%d", c, calls, bytes)
	}
}

func TestCheckBPFPGWireChecks_failConnectChurn(t *testing.T) {
	summary := &bpfSummary{
		Network: []networkStat{
			{Role: "processor", Dport: 5432, Connects: 100},
		},
	}
	checks := checkBPFPGWireChecks(summary)
	if len(checks) == 0 {
		t.Fatal("expected pg wire checks")
	}
	if checks[0].OK {
		t.Fatal("expected fail on excessive pg connects")
	}
}
