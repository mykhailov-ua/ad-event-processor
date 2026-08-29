package ingest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConversionDatacenterIPChecker_anonymousAndASN(t *testing.T) {
	table := NewDCASNTable()
	table.Publish(buildDCASNSnapshot(map[uint32]struct{}{16509: {}}, 1))
	geo := &MockGeoProvider{
		ASN: map[string]uint32{"54.230.17.9": 16509},
	}
	checker := NewConversionDatacenterIPChecker(geo, table)
	require.True(t, checker.IsDatacenterIP("1.1.1.66"))
	require.True(t, checker.IsDatacenterIP("54.230.17.9"))
	require.False(t, checker.IsDatacenterIP("8.8.8.8"))
}
