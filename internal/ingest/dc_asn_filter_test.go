package ingest

import (
	"context"
	"errors"
	"testing"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDCASNTable_mobileDenylist(t *testing.T) {
	table := NewDCASNTable()
	table.Publish(buildDCASNSnapshot(map[uint32]struct{}{
		3215:  {},
		12322: {},
		16509: {},
	}, 1))

	assert.False(t, table.IsDatacenter(3215))
	assert.False(t, table.IsDatacenter(12322))
	assert.True(t, table.IsDatacenter(16509))
}

func TestParseASNLine(t *testing.T) {
	asn, ok := parseASNLine("AS16509")
	require.True(t, ok)
	assert.Equal(t, uint32(16509), asn)
}

func TestFraudFilter_DCASN_positive(t *testing.T) {
	beforeCheck := testutil.ToFloat64(metrics.DCASNCheckTotal)
	beforeMatch := testutil.ToFloat64(metrics.DCASNMatchTotal)

	table := NewDCASNTable()
	table.Publish(buildDCASNSnapshot(map[uint32]struct{}{16509: {}}, 1))

	geo := &MockGeoProvider{ASN: map[string]uint32{"54.230.17.9": 16509}}
	f := NewFraudFilter(geo)
	f.ConfigureDCASN(table, geo, -1)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	evt.IP = "54.230.17.9"

	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.has(FraudReasonDatacenterIP))
	assert.Equal(t, beforeCheck+1, testutil.ToFloat64(metrics.DCASNCheckTotal))
	assert.Equal(t, beforeMatch+1, testutil.ToFloat64(metrics.DCASNMatchTotal))
}

func TestFraudFilter_DCASN_mobileAS3215_negative(t *testing.T) {
	table := NewDCASNTable()
	table.Publish(buildDCASNSnapshot(map[uint32]struct{}{3215: {}, 16509: {}}, 1))

	geo := &MockGeoProvider{ASN: map[string]uint32{"10.0.0.1": 3215}}
	f := NewFraudFilter(geo)
	f.ConfigureDCASN(table, geo, -1)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	evt.IP = "10.0.0.1"

	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.has(FraudReasonDatacenterIP))
}

func TestFraudFilter_DCASN_mobileAS12322_negative(t *testing.T) {
	table := NewDCASNTable()
	table.Publish(buildDCASNSnapshot(map[uint32]struct{}{12322: {}}, 1))

	geo := &MockGeoProvider{ASN: map[string]uint32{"10.0.0.2": 12322}}
	f := NewFraudFilter(geo)
	f.ConfigureDCASN(table, geo, -1)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	evt.IP = "10.0.0.2"

	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.has(FraudReasonDatacenterIP))
}

func TestFraudFilter_DCASN_sampledSkips(t *testing.T) {
	table := NewDCASNTable()
	table.Publish(buildDCASNSnapshot(map[uint32]struct{}{16509: {}}, 1))

	geo := &anonErrGeoProvider{MockGeoProvider: MockGeoProvider{ASN: map[string]uint32{"54.230.17.9": 16509}}}
	f := NewFraudFilter(geo)
	f.ConfigureDCASN(table, geo, 0)

	evt := &domain.Event{IP: "54.230.17.9", StringBuffer: make([]byte, 0, 32)}
	engine := NewFilterEngine(0, f)
	engine.SetRegistry(&mockRegistry{})

	var shadowCount int
	for i := range 256 {
		evt.FraudReason = ""
		evt.FraudScore = 0
		evt.ShadowEvent = false
		evt.StringBuffer = evt.StringBuffer[:0]
		require.NoError(t, engine.Check(context.Background(), evt))
		if evt.ShadowEvent || evt.FraudScore > 0 {
			shadowCount++
		}
	}
	assert.Greater(t, shadowCount, 0)
	assert.Less(t, shadowCount, 256)
}

func TestFraudFilter_DCASN_holdout(t *testing.T) {
	table := NewDCASNTable()
	table.Publish(buildDCASNSnapshot(map[uint32]struct{}{16509: {}}, 1))

	geo := &MockGeoProvider{ASN: map[string]uint32{"54.230.17.9": 16509}}
	f := NewFraudFilter(geo)
	f.ConfigureDCASN(table, geo, 127)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	evt.IP = "54.230.17.9"

	for range 32 {
		evt.FraudReason = ""
		evt.FraudScore = 0
		evt.StringBuffer = evt.StringBuffer[:0]
		require.NoError(t, f.Check(context.Background(), evt))
		assert.True(t, acc.has(FraudReasonDatacenterIP), "non-anonymous hosting ASN must always flag datacenter_ip")
	}
}

type anonErrGeoProvider struct {
	MockGeoProvider
}

func (p *anonErrGeoProvider) IsAnonymous(string) (bool, error) {
	return false, errors.New("geo unavailable")
}

func TestFraudFilter_DCASN_engineL2Shadow(t *testing.T) {
	table := NewDCASNTable()
	table.Publish(buildDCASNSnapshot(map[uint32]struct{}{16509: {}}, 1))
	geo := &MockGeoProvider{ASN: map[string]uint32{"54.230.17.9": 16509}}
	f := NewFraudFilter(geo)
	f.ConfigureDCASN(table, geo, -1)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	evt.CampaignID = uuid.New()
	evt.IP = "54.230.17.9"

	engine := NewFilterEngine(0, f)
	engine.SetRegistry(&mockRegistry{})
	require.NoError(t, engine.Check(context.Background(), evt))
	assert.Contains(t, evt.FraudReason, FraudReasonCodeDatacenterIP)
	assert.True(t, evt.ShadowEvent)
}

func TestDCASNMetrics_registered(t *testing.T) {
	require.NotNil(t, metrics.DCASNCheckTotal)
	require.NotNil(t, metrics.DCASNMatchTotal)
}
