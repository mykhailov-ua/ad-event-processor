package licensing_test

import (
	"testing"

	"ad-event-processor/internal/licensing"
)

func BenchmarkStretchMCKForRecheck(b *testing.B) {
	var mck [32]byte
	mck[0] = 0xba
	deploymentID := "dep-mck-vector"
	for b.Loop() {
		_, err := licensing.StretchMCKForRecheck(mck, deploymentID)
		if err != nil {
			b.Fatal(err)
		}
	}
}
