package testutil

import (
	"testing"

	"github.com/bidshard/ad-event-processor/pkg/faultproof"
)

func LogFaultProof(t testing.TB, scenario string, attrs map[string]string) {
	t.Helper()
	faultproof.Log(t, scenario, attrs)
}
