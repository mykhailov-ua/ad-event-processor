package controlplane

import (
	"strings"
	"testing"

	"github.com/bidshard/ad-event-processor/pkg/naming"
	"github.com/stretchr/testify/assert"
)

func TestParseEdgePrometheus_canonicalNames(t *testing.T) {
	body := strings.Join([]string{
		"ad_event_processor_edge_ingress_protocol_h1_total 10",
		"ad_event_processor_edge_body_read_total 3",
		"ad_event_processor_edge_blocked_ip_total 7",
		"ad_event_processor_edge_tarpit_total 2",
	}, "\n")
	out := parseEdgePrometheus(strings.NewReader(body))
	assert.Equal(t, uint64(10), out.IngressH1)
	assert.Equal(t, uint64(3), out.BodyRead)
	assert.Equal(t, uint64(7), out.Blocked["ip_blacklist"])
	assert.Equal(t, uint64(2), out.TarpitTotal)
}

func TestParseEdgePrometheus_legacyAlias(t *testing.T) {
	body := naming.DeprecatedEdgeMetricPrefix() + "blocked_campaign_rl_total 5\n"
	out := parseEdgePrometheus(strings.NewReader(body))
	assert.Equal(t, uint64(5), out.Blocked["campaign_rl"])
}
