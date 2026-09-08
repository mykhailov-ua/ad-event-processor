package filter

import (
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cgnatTestASNLookup struct {
	asn uint32
	ok  bool
}

func (l cgnatTestASNLookup) LookupASN(string) (uint32, bool) {
	return l.asn, l.ok
}

func TestCGNAT_holdoutBlacklistBypassMobileCarrier(t *testing.T) {
	carrier := NewMobileCarrierASNTable(nil)
	lookup := cgnatTestASNLookup{asn: 21928, ok: true}
	camp := &domain.Campaign{CgnatIPPolicyEnabled: true}
	evt := &domain.Event{}

	require.True(t, ShouldBypassCGNATIPBlacklist(false, camp, carrier, lookup, "10.1.2.3", evt))
}

func TestCGNAT_holdoutBlacklistBlockedWithProbeCluster(t *testing.T) {
	carrier := NewMobileCarrierASNTable(nil)
	lookup := cgnatTestASNLookup{asn: 21928, ok: true}
	camp := &domain.Campaign{CgnatIPPolicyEnabled: true}
	evt := &domain.Event{ProbeClusterSet: 1, ProbeClusterRoute: 1}

	assert.False(t, ShouldBypassCGNATIPBlacklist(false, camp, carrier, lookup, "10.1.2.3", evt))
}

func TestCGNAT_holdoutBlacklistNotBypassedOffCarrier(t *testing.T) {
	carrier := NewMobileCarrierASNTable(nil)
	lookup := cgnatTestASNLookup{asn: 16509, ok: true}
	camp := &domain.Campaign{CgnatIPPolicyEnabled: true}
	evt := &domain.Event{}

	assert.False(t, ShouldBypassCGNATIPBlacklist(false, camp, carrier, lookup, "54.230.17.9", evt))
}
