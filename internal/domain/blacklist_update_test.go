package domain

import "testing"

func TestDefaultBlacklistUpdateChannel(t *testing.T) {
	if got := DefaultBlacklistUpdateChannel(""); got != BlacklistUpdateChannel {
		t.Fatalf("DefaultBlacklistUpdateChannel(\"\")=%q want %q", got, BlacklistUpdateChannel)
	}
	if got := DefaultBlacklistUpdateChannel("custom"); got != "custom" {
		t.Fatalf("DefaultBlacklistUpdateChannel(custom)=%q want custom", got)
	}
}
