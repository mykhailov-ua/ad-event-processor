package verify_test

import (
	"testing"

	"ad-event-processor/internal/licensing/verify"
)

func BenchmarkStretchMCKForRecheck(b *testing.B) {
	var mck [32]byte
	mck[0] = 0xba
	deploymentID := "dep-mck-vector"
	for b.Loop() {
		_, err := verify.StretchMCKForRecheck(mck, deploymentID)
		if err != nil {
			b.Fatal(err)
		}
	}
}
