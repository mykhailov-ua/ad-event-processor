package reports

import "testing"

func TestComputeCPCMicro(t *testing.T) {
	if got := ComputeCPCMicro(2_000_000, 100); got != 20_000 {
		t.Fatalf("ComputeCPCMicro() = %d, want 20000", got)
	}
	if got := ComputeCPCMicro(0, 100); got != 0 {
		t.Fatalf("ComputeCPCMicro zero cost = %d, want 0", got)
	}
}

func TestComputeCRPct(t *testing.T) {
	if got := ComputeCRPct(25, 1000); got != 2.5 {
		t.Fatalf("ComputeCRPct() = %v, want 2.5", got)
	}
}

func TestComputeROIPct(t *testing.T) {
	if got := ComputeROIPct(500_000, 2_000_000); got != 25 {
		t.Fatalf("ComputeROIPct() = %v, want 25", got)
	}
}

func TestEnrichBreakdownEconomics(t *testing.T) {
	row := DashboardBreakdownRowDTO{
		Clicks:       200,
		Conversions:  10,
		CostMicro:    4_000_000,
		RevenueMicro: 5_000_000,
	}
	EnrichBreakdownEconomics(&row)
	if row.ProfitMicro != 1_000_000 {
		t.Fatalf("ProfitMicro = %d, want 1000000", row.ProfitMicro)
	}
	if row.CPCMicro != 20_000 {
		t.Fatalf("CPCMicro = %d, want 20000", row.CPCMicro)
	}
	if row.CPAMicro != 400_000 {
		t.Fatalf("CPAMicro = %d, want 400000", row.CPAMicro)
	}
	if row.CRPct != 5 {
		t.Fatalf("CRPct = %v, want 5", row.CRPct)
	}
	if row.EPCMicro != 25_000 {
		t.Fatalf("EPCMicro = %d, want 25000", row.EPCMicro)
	}
}
