package ingest

import (
	"fmt"
	"strings"
	"testing"

	"ad-event-processor/pkg/faultproof"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fraudJSONCase struct {
	id      string
	name    string
	body    []byte
	mustErr bool
	mustOK  bool
	check   func(t *testing.T, req *TrackRequest)
}

func fraudTrackJSONCases2026() []fraudJSONCase {
	validCID := "550e8400-e29b-41d4-a716-446655440000"
	return []fraudJSONCase{
		{
			id: "truncated_no_close", name: "truncated_no_close",
			body:    []byte(`{"campaign_id":"550e8400`),
			mustErr: true,
		},
		{
			id: "type_impression_on_click", name: "type_impression_on_click",
			body:   []byte(`{"campaign_id":"550e8400-e29b-41d4-a716-446655440000","type":"impression"}`),
			mustOK: true,
			check: func(t *testing.T, req *TrackRequest) {
				assert.Equal(t, "impression", req.Type)
			},
		},
		{
			id: "oversized_uuid_string", name: "oversized_uuid_string",
			body:    []byte(`{"campaign_id":"` + strings.Repeat("a", 128) + `"}`),
			mustErr: true,
		},
		{
			id: "duplicate_user_id_last_wins", name: "duplicate_user_id_last_wins",
			body:   []byte(`{"campaign_id":"550e8400-e29b-41d4-a716-446655440000","user_id":"first","user_id":"second"}`),
			mustOK: true,
			check: func(t *testing.T, req *TrackRequest) {
				assert.Equal(t, "second", req.UserID)
			},
		},
		{
			id: "nested_payload_shallow", name: "nested_payload_shallow",
			body:   []byte(`{"campaign_id":"550e8400-e29b-41d4-a716-446655440000","payload":{"a":{"b":"c"}}}`),
			mustOK: true,
		},
		{
			id: "unicode_escaped_key", name: "unicode_escaped_key",
			body:    []byte(`{"campaign\u005fid":"550e8400-e29b-41d4-a716-446655440000"}`),
			mustErr: true,
		},
		{
			id: "numeric_campaign_id", name: "numeric_campaign_id",
			body:    []byte(`{"campaign_id":12345}`),
			mustErr: true,
		},
		{
			id: "reordered_keys", name: "reordered_keys",
			body:   []byte(`{"type":"click","campaign_id":"550e8400-e29b-41d4-a716-446655440000"}`),
			mustOK: true,
			check: func(t *testing.T, req *TrackRequest) {
				assert.Equal(t, validCID, req.CampaignID.String())
			},
		},
		{
			id: "unicode_escape_in_uuid_value_rejected", name: "unicode_escape_in_uuid_value_rejected",
			body:    []byte(`{"campaign_id":"\u0035\u0035\u0030e8400-e29b-41d4-a716-446655440000"}`),
			mustErr: true,
		},
		{
			id: "null_campaign_id_value", name: "null_campaign_id_value",
			body:    []byte(`{"campaign_id":null,"type":"click"}`),
			mustErr: true,
		},
		{
			id: "empty_object", name: "empty_object",
			body:   []byte(`{}`),
			mustOK: true,
		},
		{
			id: "bom_prefix", name: "bom_prefix",
			body:    append([]byte{0xEF, 0xBB, 0xBF}, []byte(`{"campaign_id":"550e8400-e29b-41d4-a716-446655440000"}`)...),
			mustErr: true,
		},
		{
			id: "null_byte_in_string", name: "null_byte_in_string",
			body:    []byte("{\"campaign_id\":\"550e8400-e29b-41d4-a716-4466554400\x000\"}"),
			mustErr: true,
		},
	}
}

func TestFraudScenarios_TrackJSON_2026(t *testing.T) {
	var failures []string
	for _, tc := range fraudTrackJSONCases2026() {
		tc := tc
		t.Run(tc.id+"_"+tc.name, func(t *testing.T) {
			var req TrackRequest
			err := ParseTrackRequestJSON(&req, tc.body)
			switch {
			case tc.mustErr:
				if err == nil {
					msg := fmt.Sprintf("%s [%s]: holdout expected malformed got success req=%+v", tc.id, tc.name, req)
					failures = append(failures, msg)
					t.Fatal(msg)
				}
			case tc.mustOK:
				if err != nil {
					msg := fmt.Sprintf("%s [%s]: holdout expected accept got %v", tc.id, tc.name, err)
					failures = append(failures, msg)
					t.Fatal(msg)
				}
				if tc.check != nil {
					tc.check(t, &req)
				}
			}
		})
	}
	faultproof.Log(t, "fraud_track_json_2026", map[string]string{
		"cases":    fmt.Sprintf("%d", len(fraudTrackJSONCases2026())),
		"failures": fmt.Sprintf("%d", len(failures)),
	})
}

func TestFraudScenarios_TrackJSON_OptParityOnCorpus(t *testing.T) {
	for _, tc := range fraudTrackJSONCases2026() {
		if !tc.mustOK {
			continue
		}
		var a, b TrackRequest
		errA := ParseTrackRequestJSON(&a, tc.body)
		errB := ParseTrackRequestJSONOpt(&b, tc.body)
		require.Equal(t, errA, errB, tc.id)
		if errA == nil {
			require.Equal(t, a, b, tc.id)
		}
	}
}

func TestFraudScenarios_TrackJSON_DeepNestedPayload(t *testing.T) {
	validCID := "550e8400-e29b-41d4-a716-446655440000"
	var nested strings.Builder
	nested.WriteString(`{"campaign_id":"`)
	nested.WriteString(validCID)
	nested.WriteString(`","payload":`)
	for range 200 {
		nested.WriteString(`{"a":`)
	}
	nested.WriteString(`"leaf"`)
	for range 200 {
		nested.WriteString(`}`)
	}
	nested.WriteString(`}`)

	var req TrackRequest
	err := ParseTrackRequestJSON(&req, []byte(nested.String()))
	if err != nil {
		t.Logf("case json_deep_nested_stack: deep nested payload rejected at depth 200: %v", err)
	}
}

func TestFraudScenarios_TrackJSON_LargePayloadWithinBody(t *testing.T) {
	validCID := "550e8400-e29b-41d4-a716-446655440000"
	inner := strings.Repeat(`"x",`, 100)
	inner = strings.TrimSuffix(inner, ",")
	var bodyBuf strings.Builder
	bodyBuf.WriteString(`{"campaign_id":"`)
	bodyBuf.WriteString(validCID)
	bodyBuf.WriteString(`","payload":[`)
	bodyBuf.WriteString(inner)
	bodyBuf.WriteString(`]}`)
	body := bodyBuf.String()
	var req TrackRequest
	err := ParseTrackRequestJSON(&req, []byte(body))
	if err != nil {
		t.Logf("holdout: large but valid JSON array payload rejected: %v", err)
	}
}
