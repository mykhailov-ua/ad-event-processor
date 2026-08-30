package ingest

import (
	"context"
	"net/http"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMobileCarrierASNTable_builtinASN(t *testing.T) {
	t.Parallel()
	table := NewMobileCarrierASNTable(nil)
	require.True(t, table.IsMobileCarrier(21928))
	require.False(t, table.IsMobileCarrier(16509))
}

func TestCGNAT_mobileASN_skipsIPv4Rotation_holdout(t *testing.T) {
	filter := &countingFilter{}
	h, cid := ipv4RotationHandler(t, true, "live", 2, filter)
	h.ConfigureMobileCarrierASN(NewMobileCarrierASNTable(nil))
	h.cfg.CGNATMobileIPBypass = true
	asnMap := make(map[string]uint32, 4)
	for i := 1; i <= 4; i++ {
		asnMap[rotatedIPv4Host(i)] = 21928
	}
	h.trackProc.ingestGeo = &MockGeoProvider{ASN: asnMap}

	for i := 1; i <= 4; i++ {
		conn := serveClickFromIPUser(h, cid, rotatedIPv4Host(i), "user-cgnat")
		require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: l1v4")
	}
}

func TestCGNAT_mobileASN_ipv4Rotation_holdout_withoutBypass(t *testing.T) {
	filter := &countingFilter{}
	h, cid := ipv4RotationHandler(t, true, "live", 3, filter)
	h.ConfigureMobileCarrierASN(NewMobileCarrierASNTable(nil))
	asnMap := make(map[string]uint32, 4)
	for i := 1; i <= 4; i++ {
		asnMap[rotatedIPv4Host(i)] = 21928
	}
	h.trackProc.ingestGeo = &MockGeoProvider{ASN: asnMap}

	for i := 1; i <= 2; i++ {
		conn := serveClickFromIPUser(h, cid, rotatedIPv4Host(i), "user-cgnat")
		require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: l1v4")
	}
	conn := serveClickFromIPUser(h, cid, rotatedIPv4Host(3), "user-cgnat")
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	require.Contains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: l1v4")
}

func TestCGNAT_mobileASN_stillBlocksDatacenter(t *testing.T) {
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
	evt.Type = "click"
	evt.CampaignID = uuid.New()
	require.NoError(t, f.Check(context.Background(), evt))
	require.True(t, acc.Has(FraudReasonDatacenterIP))

	carrier := NewMobileCarrierASNTable(nil)
	require.False(t, carrier.IsMobileCarrier(16509))
}

func TestCGNAT_bypassIngressRPD(t *testing.T) {
	camp := &domain.Campaign{CgnatIPPolicyEnabled: true}
	carrier := NewMobileCarrierASNTable(nil)
	geo := &MockGeoProvider{ASN: map[string]uint32{"10.1.2.3": 21928}}
	require.True(t, shouldBypassCGNATIPVelocity(false, camp, carrier, geo, "10.1.2.3", "ingress_rpd"))
}
