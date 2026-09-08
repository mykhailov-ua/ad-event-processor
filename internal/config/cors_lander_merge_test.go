package config

import "testing"

func TestMergeTrackCORSOriginsWithLander_appendsLanderOrigin(t *testing.T) {
	cfg := &Config{
		TrackCORSOrigins:    []string{"https://trk.example.com"},
		LanderPublicBaseURL: "https://lp.example.com",
	}
	mergeTrackCORSOriginsWithLander(cfg)
	if len(cfg.TrackCORSOrigins) != 2 {
		t.Fatalf("origins %v", cfg.TrackCORSOrigins)
	}
	if cfg.TrackCORSOrigins[1] != "https://lp.example.com" {
		t.Fatalf("got %q", cfg.TrackCORSOrigins[1])
	}
}

func TestMergeTrackCORSOriginsWithLander_skipsDuplicate(t *testing.T) {
	cfg := &Config{
		TrackCORSOrigins:    []string{"https://lp.example.com"},
		LanderPublicBaseURL: "https://lp.example.com",
	}
	mergeTrackCORSOriginsWithLander(cfg)
	if len(cfg.TrackCORSOrigins) != 1 {
		t.Fatalf("origins %v", cfg.TrackCORSOrigins)
	}
}
