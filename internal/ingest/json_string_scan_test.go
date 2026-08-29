package ingest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseQuotedField_tidRegression(t *testing.T) {
	w := []byte(`"tid":"supply-txn-42"}`)
	var buf [128]byte
	ln := parseQuotedField(w, len(`"tid"`), buf[:])
	require.Equal(t, 13, ln)
	require.Equal(t, "supply-txn-42", string(buf[:ln]))
}

func TestScanJSONStringEnd_truncatedSurrogatePairNoPanic(t *testing.T) {
	const payload = `"\uD800\uDC"`
	bud := newJSONScanBudget()
	_, ok := scanJSONStringEnd([]byte(payload), 0, len(payload), &bud)
	require.False(t, ok)
}
