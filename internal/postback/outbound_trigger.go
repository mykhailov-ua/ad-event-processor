package postback

import (
	"encoding/json"
	"strings"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
)

const (
	OutboundTriggerConversion = "conversion"
	OutboundTriggerStatus     = "status"
	OutboundTriggerGoal       = "goal"
)

func OutboundPostbackMatchesEvent(row db.CampaignOutboundPostback, evt *domain.Event) bool {
	if evt == nil || !row.Enabled {
		return false
	}
	kind := strings.ToLower(strings.TrimSpace(row.TriggerKind))
	if kind == "" {
		kind = OutboundTriggerConversion
	}
	switch kind {
	case OutboundTriggerConversion:
		return eventTypeMatches(evt.Type, row.TargetEvent)
	case OutboundTriggerStatus:
		return triggerValueMatchesPayload(evt.Payload, row.TriggerValue, statusPayloadKeys)
	case OutboundTriggerGoal:
		return triggerValueMatchesPayload(evt.Payload, row.TriggerValue, []string{"goal_name"})
	default:
		return false
	}
}

var statusPayloadKeys = []string{"status", "affiliate_status", "conversion_status", "lead_status", "internal_status"}

func triggerValueMatchesPayload(payload []byte, want string, keys []string) bool {
	want = normalizeTriggerValue(want)
	if want == "" {
		return false
	}
	fields := readPayloadStringFields(payload)
	for _, key := range keys {
		if normalizeTriggerValue(fields[key]) == want {
			return true
		}
	}
	return false
}

func normalizeTriggerValue(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func readPayloadStringFields(payload []byte) map[string]string {
	if len(payload) == 0 {
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil
	}
	out := make(map[string]string, len(raw))
	for key, val := range raw {
		if len(val) == 0 {
			continue
		}
		if val[0] == '"' {
			var s string
			if json.Unmarshal(val, &s) == nil {
				out[key] = s
			}
			continue
		}
		out[key] = strings.TrimSpace(string(val))
	}
	return out
}

func PostbackConfigFromOutbound(row db.CampaignOutboundPostback) db.PostbackConfig {
	return db.PostbackConfig{
		CampaignID:             row.CampaignID,
		Provider:               row.Provider,
		UrlTemplate:            row.UrlTemplate,
		ApiTokenEncrypted:      row.ApiTokenEncrypted,
		TargetEvent:            row.TargetEvent,
		TestEventCode:          row.TestEventCode,
		SigningSecretEncrypted: row.SigningSecretEncrypted,
	}
}
