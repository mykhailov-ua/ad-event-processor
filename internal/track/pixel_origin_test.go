package track

import (
	"strings"
	"testing"
)

func TestResolveBrowserPixel_prefersLanderFirstPartyPath(t *testing.T) {
	bundle := ResolveBrowserPixel(
		"https://lp.example.com",
		"https://trk.example.com",
		"550e8400-e29b-41d4-a716-446655440000",
	)
	if !bundle.FirstParty {
		t.Fatal("expected first-party")
	}
	if bundle.ScriptURL != "https://lp.example.com/_aed/track.js" {
		t.Fatalf("script url %q", bundle.ScriptURL)
	}
	if bundle.TrackURL != "https://trk.example.com/track" {
		t.Fatalf("track url %q", bundle.TrackURL)
	}
	if !strings.Contains(bundle.Snippet, "/_aed/track.js") {
		t.Fatalf("snippet %q", bundle.Snippet)
	}
	if !strings.Contains(bundle.Snippet, "conversionEventId") {
		t.Fatal("missing conversionEventId")
	}
}

func TestResolveBrowserPixel_trackerStaticWithoutLander(t *testing.T) {
	bundle := ResolveBrowserPixel("", "trk.example.com", "550e8400-e29b-41d4-a716-446655440000")
	if bundle.FirstParty {
		t.Fatal("unexpected first-party")
	}
	if bundle.ScriptURL != "https://trk.example.com/static/track.js" {
		t.Fatalf("script url %q", bundle.ScriptURL)
	}
}
