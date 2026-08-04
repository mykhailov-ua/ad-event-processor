package doctor

import "testing"

func TestCheckHint(t *testing.T) {
	if got := CheckHint("redis"); got == "" {
		t.Fatal("expected redis hint")
	}
	if got := CheckHint("unknown_probe"); got != "" {
		t.Fatalf("unexpected hint: %q", got)
	}
}

func TestReportToDTOIncludesHint(t *testing.T) {
	rep := Report{
		Results: []Result{
			{Name: "redis", Status: StatusFail, Detail: "ping failed"},
		},
	}
	dto := ReportToDTO(rep)
	if len(dto) != 1 {
		t.Fatalf("len=%d", len(dto))
	}
	if dto[0].Hint == "" {
		t.Fatal("expected hint on redis check")
	}
}
