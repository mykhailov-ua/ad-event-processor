package ingest

import (
	"testing"
)

func FuzzSkipJSONValueBudget(f *testing.F) {
	f.Add([]byte(`{"a":1}`), byte(16))
	f.Add([]byte(`[1,2,3]`), byte(8))
	f.Add([]byte(`"x"`), byte(32))

	f.Fuzz(func(t *testing.T, data []byte, maxDepth uint8) {
		if maxDepth == 0 {
			maxDepth = 1
		}
		bud := newJSONScanBudget()
		fuzzNoPanic(t, "skipJSONValueBudgetDepth", func() {
			_, _ = skipJSONValueBudgetDepth(data, 0, &bud, int(maxDepth))
		})
	})
}

func FuzzScanJSONStringEnd(f *testing.F) {
	f.Add([]byte(`"abc"`))
	f.Add([]byte(`"\u0041"`))
	f.Add([]byte(`"\\"`))

	f.Fuzz(func(t *testing.T, data []byte) {
		bud := newJSONScanBudget()
		fuzzNoPanic(t, "scanJSONStringEnd", func() {
			_, _ = scanJSONStringEnd(data, 0, len(data), &bud)
		})
		fuzzNoPanic(t, "scanJSONLiteralStringEnd", func() {
			_, _ = scanJSONLiteralStringEnd(data, 0, len(data), &bud)
		})
	})
}
