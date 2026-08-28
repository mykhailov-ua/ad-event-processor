package doctor

import (
	"context"
	"testing"
	"time"
)

type stubProbe struct {
	name   string
	result Result
}

func (p stubProbe) Name() string { return p.name }
func (p stubProbe) Run(context.Context) Result {
	return p.result
}

func TestRunOnlyFilter(t *testing.T) {
	probes := []Probe{
		stubProbe{name: "redis", result: Result{Name: "redis", Status: StatusPass}},
		stubProbe{name: "sysctl", result: Result{Name: "sysctl", Status: StatusPass}},
	}
	rep := Run(context.Background(), Options{
		Only:   []string{"redis"},
		Probes: probes,
	})
	if len(rep.Results) != 1 || rep.Results[0].Name != "redis" {
		t.Fatalf("expected only redis probe, got %+v", rep.Results)
	}
}

func TestReportExitCode(t *testing.T) {
	tests := []struct {
		name string
		rep  Report
		want int
	}{
		{"all pass", Report{Results: []Result{{Status: StatusPass}, {Status: StatusSkip}}}, 0},
		{"warn", Report{Results: []Result{{Status: StatusWarn}}}, 1},
		{"fail", Report{Results: []Result{{Status: StatusFail}}}, 2},
		{"fail beats warn", Report{Results: []Result{{Status: StatusWarn}, {Status: StatusFail}}}, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.rep.ExitCode(); got != tc.want {
				t.Fatalf("ExitCode()=%d want %d", got, tc.want)
			}
		})
	}
}

func TestRunRespectsTimeout(t *testing.T) {
	slow := stubProbe{
		name:   "slow",
		result: Result{Name: "slow", Status: StatusPass},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	_ = Run(ctx, Options{Timeout: 1 * time.Nanosecond, Probes: []Probe{slow}})
}

func TestDSNSSLMode(t *testing.T) {
	mode, err := dsnSSLMode("postgres://user:pass@localhost/db?sslmode=verify-full")
	if err != nil {
		t.Fatal(err)
	}
	if mode != "verify-full" {
		t.Fatalf("mode=%q", mode)
	}
	mode, err = dsnSSLMode("postgres://localhost/db")
	if err != nil {
		t.Fatal(err)
	}
	if mode != "disable" {
		t.Fatalf("mode=%q", mode)
	}
}
