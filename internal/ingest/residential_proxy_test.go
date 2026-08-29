package ingest

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type policyParityFile struct {
	Cases []policyParityCase `json:"cases"`
}

type policyParityCase struct {
	ID   string          `json:"id"`
	Op   string          `json:"op"`
	Row  policyParityRow `json:"row"`
	Want json.RawMessage `json:"want"`
}

type policyParityRow struct {
	Events      int `json:"events"`
	Clicks      int `json:"clicks"`
	UniqueUsers int `json:"unique_users"`
	UniqueUAs   int `json:"unique_uas"`
}

func modelPolicyParityPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(file), "..", "..", "model", "testdata", "policy_parity.json")
}

func TestResidentialProxyPolicyParity(t *testing.T) {
	path := modelPolicyParityPath(t)
	require.FileExists(t, path)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var file policyParityFile
	require.NoError(t, json.Unmarshal(data, &file))
	require.NotEmpty(t, file.Cases)

	cfg := DefaultResidentialProxyPolicyForTest()
	for _, tc := range file.Cases {
		if tc.Op != "residential_proxy_signal" {
			continue
		}
		t.Run(tc.ID, func(t *testing.T) {
			var want bool
			require.NoError(t, json.Unmarshal(tc.Want, &want))
			row := ResidentialProxyRow{
				Events:      tc.Row.Events,
				Clicks:      tc.Row.Clicks,
				UniqueUsers: tc.Row.UniqueUsers,
				UniqueUAs:   tc.Row.UniqueUAs,
			}
			assert.Equal(t, want, ResidentialProxySignalForTest(row, cfg))
		})
	}
}

func TestResidentialProxySignal_holdoutPositive(t *testing.T) {
	cfg := DefaultResidentialProxyPolicyForTest()
	row := ResidentialProxyRow{Events: 275, Clicks: 4, UniqueUsers: 32, UniqueUAs: 11}
	require.True(t, ResidentialProxySignalForTest(row, cfg))
}

func TestResidentialProxyFilter_positiveFarm(t *testing.T) {
	before := testutil.ToFloat64(metrics.ResidentialProxySignalTotal)
	ring := NewResidentialProxyRing()
	f := NewResidentialProxyFilter(ring)
	cid := uuid.New()
	ring.SeedForTest(cid, ResidentialProxyRow{
		Events:      275,
		Clicks:      4,
		UniqueUsers: 32,
		UniqueUAs:   11,
	})

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	evt.CampaignID = cid
	evt.Type = "impression"
	evt.UserID = "user-final"
	evt.UA = "ua-final"

	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.has(FraudReasonResidentialProxy))
	assert.Equal(t, before+1, testutil.ToFloat64(metrics.ResidentialProxySignalTotal))
}

func TestResidentialProxyFilter_negativeHighCTR(t *testing.T) {
	ring := NewResidentialProxyRing()
	f := NewResidentialProxyFilter(ring)
	cid := uuid.New()
	campaignHash := CRC32Castagnoli(&cid)
	now := monotonicNano()

	for i := range 900 {
		ring.Observe(campaignHash, false, HashResidentialProxyUser("u"+itoaResidential(i)), HashResidentialProxyUA("ua"+itoaResidential(i%70)), now)
	}
	for i := range 120 {
		ring.Observe(campaignHash, true, HashResidentialProxyUser("c"+itoaResidential(i)), HashResidentialProxyUA("ua"+itoaResidential(i%70)), now)
	}

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	evt.CampaignID = cid
	evt.Type = "click"
	evt.UserID = "extra"
	evt.UA = "ua-extra"

	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.has(FraudReasonResidentialProxy))
}

func itoaResidential(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
