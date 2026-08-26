package main

import "testing"

func TestLoadgenProcessMatch_comm(t *testing.T) {
	t.Parallel()
	if !loadgenProcessMatch("loadgen", "", []string{"loadgen"}) {
		t.Fatal("expected comm match")
	}
	if loadgenProcessMatch("tracker", "", []string{"loadgen"}) {
		t.Fatal("unexpected comm match")
	}
}

func TestLoadgenProcessMatch_cmdlineGoRun(t *testing.T) {
	t.Parallel()
	cmdline := "/usr/local/go/bin/go run ./cmd/loadgen -mode smoke -out /tmp/out"
	if !loadgenProcessMatch("go", cmdline, []string{"loadgen"}) {
		t.Fatal("expected cmd/loadgen match for go run")
	}
}

func TestLoadgenProcessMatch_cmdlineBinary(t *testing.T) {
	t.Parallel()
	cmdline := "/home/dev/ad-event-processor/bin/loadgen -mode business -out /tmp/out"
	if !loadgenProcessMatch("loadgen", cmdline, []string{"loadgen"}) {
		t.Fatal("expected bin/loadgen match")
	}
}
