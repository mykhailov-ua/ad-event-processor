package trialregistry

import (
	"os"
	"testing"
)

func TestSelfServeURL_fromEnv(t *testing.T) {
	t.Setenv(EnvTrialSelfServeURL, "https://t.me/example_bot")
	if got := SelfServeURL(); got != "https://t.me/example_bot" {
		t.Fatalf("got %q", got)
	}
}

func TestSelfServeURL_emptyWhenUnset(t *testing.T) {
	_ = os.Unsetenv(EnvTrialSelfServeURL)
	if got := SelfServeURL(); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}
