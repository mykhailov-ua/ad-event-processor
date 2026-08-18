package ingestion

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"github.com/stretchr/testify/require"
)

func TestFault_SecurityGap_H1_04b_ObsFoldContinuation(t *testing.T) {
	payload := []byte("POST /track HTTP/1.1\r\nX-Evil: safe\r\n continuation\r\nContent-Length: 0\r\n\r\n")
	_, _, err := parseHTTP1(payload, 1024, nil)

	disposition := "rejected"
	gap := "closed"
	if err == nil {
		disposition = "accepted"
		gap = "open"
	}

	faultproof.Log(t, "security_gap_h1_04b", map[string]string{
		"gap_id":      "H1-04b",
		"gap":         gap,
		"disposition": disposition,
		"risk":        "header_injection_obs_fold",
		"err":         fmt.Sprintf("%v", err),
	})
	require.Equal(t, "closed", gap)
}

func TestFault_SecurityGap_XDFA_04_ProtoAsJSON(t *testing.T) {
	proto := testProtoTrackBody(t)
	wire := append([]byte(fmt.Sprintf("POST /track HTTP/1.1\r\nContent-Length: %d\r\n\r\n", len(proto))), proto...)
	n, req, err := parseHTTP1(wire, 1<<20, nil)
	require.NoError(t, err)
	require.Equal(t, len(wire), n)

	var tr TrackRequest
	parseErr := ParseTrackRequestJSON(&tr, req.Body)

	disposition := "rejected"
	gap := "closed"
	if parseErr == nil {
		disposition = "accepted"
		gap = "open"
	}

	faultproof.Log(t, "security_gap_xdfa_04", map[string]string{
		"gap_id":      "XDFA-04",
		"gap":         gap,
		"disposition": disposition,
		"risk":        "content_type_confusion_proto_json",
		"parse_err":   fmt.Sprintf("%v", parseErr),
	})
	require.Equal(t, "closed", gap)
}

func TestFault_SecurityGap_G_J05b_DeepNestedJSON(t *testing.T) {
	const depth = 200
	validCID := "550e8400-e29b-41d4-a716-446655440000"
	var nested strings.Builder
	nested.WriteString(`{"campaign_id":"`)
	nested.WriteString(validCID)
	nested.WriteString(`","payload":`)
	for range depth {
		nested.WriteString(`{"a":`)
	}
	nested.WriteString(`"leaf"`)
	for range depth {
		nested.WriteString(`}`)
	}
	nested.WriteString(`}`)

	var tr TrackRequest
	parseErr := ParseTrackRequestJSON(&tr, []byte(nested.String()))
	require.Error(t, parseErr, "deep nested JSON must be rejected")
	require.ErrorIs(t, parseErr, ErrMalformed)

	faultproof.Log(t, "security_gap_g_j05b", map[string]string{
		"gap_id":      "G-J05b",
		"gap":         "closed",
		"disposition": "rejected",
		"risk":        "deep_nested_json_stack",
		"depth":       fmt.Sprintf("%d", depth),
		"parse_err":   fmt.Sprintf("%v", parseErr),
	})
}
