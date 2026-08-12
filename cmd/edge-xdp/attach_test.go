package main

import "testing"

func TestXdpAttachModes_offloadFallbackChain(t *testing.T) {
	got := xdpAttachModes("offload")
	want := []string{"offload", "native", "generic"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestXdpAttachModes_nativeFallbackChain(t *testing.T) {
	got := xdpAttachModes("native")
	want := []string{"native", "generic"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestXdpAttachModes_genericOnly(t *testing.T) {
	got := xdpAttachModes("generic")
	if len(got) != 1 || got[0] != "generic" {
		t.Fatalf("got %v", got)
	}
}
