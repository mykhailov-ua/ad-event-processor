package doctor

import (
	"context"
	"testing"
)

func TestDNSProbeName(t *testing.T) {
	if (DNSProbe{}).Name() != "dns" {
		t.Fatalf("Name()=%q want dns", (DNSProbe{}).Name())
	}
}

func TestDNSProbeSkipEmptyHostname(t *testing.T) {
	r := (DNSProbe{Hostname: ""}).Run(context.Background())
	if r.Status != StatusSkip {
		t.Fatalf("Status=%v want skip", r.Status)
	}
	if r.Name != "dns" {
		t.Fatalf("Name=%q", r.Name)
	}
}

func TestDNSProbeSkipWhitespaceHostname(t *testing.T) {
	r := (DNSProbe{Hostname: " "}).Run(context.Background())
	if r.Status != StatusSkip {
		t.Fatalf("Status=%v want skip", r.Status)
	}
}
