package entitlements_test

import (
	"bytes"
	"strings"
	"testing"

	verify "ad-event-processor/internal/licensing/verify"

	"github.com/stretchr/testify/require"
)

func TestDecodeJSONStrict_MalformedAndOversized(t *testing.T) {
	var dst struct {
		PlanCode string `json:"plan_code"`
		Status   string `json:"status"`
	}
	err := verify.DecodeJSONStrict(bytes.NewReader([]byte(`{"plan_code":"basic","status":"active",}`)), 1024, &dst)
	require.Error(t, err)
	require.ErrorIs(t, err, verify.ErrJSONMalformed)

	huge := []byte(`{"plan_code":"` + strings.Repeat("a", 128*1024) + `"}`)
	err = verify.DecodeJSONStrict(bytes.NewReader(huge), 4096, &dst)
	require.ErrorIs(t, err, verify.ErrJSONTooLarge)

	err = verify.DecodeJSONStrict(bytes.NewReader([]byte(`{"plan_code":"basic","status":"active","unknown":1}`)), 1024, &dst)
	require.Error(t, err)
}
