package loadreport

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckOpenRTBSLA_naPrometheusPasses(t *testing.T) {
	result, err := CheckOpenRTBSLA("http://127.0.0.1:1")
	require.NoError(t, err)
	assert.True(t, result.Pass)
	require.Len(t, result.Checks, 2)
}

func TestCheckScalarMS(t *testing.T) {
	prom := newPromClient("http://invalid")
	chk := checkScalarMS(prom, "test", `vector(1)`, 80)
	assert.True(t, chk.OK)
}
