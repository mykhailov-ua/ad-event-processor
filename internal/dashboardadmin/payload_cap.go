package dashboardadmin

import (
	"encoding/json"
	"net/http"

	"ad-event-processor/pkg/httpresponse"
)

const maxDashboardTableRows = 100

// DashboardTableMeta describes server-side truncation for a dashboard table section.
type DashboardTableMeta struct {
	Truncated bool `json:"truncated"`
	Total     int  `json:"total"`
}

func capDashboardTableSections(payload map[string]any) map[string]any {
	if len(payload) == 0 {
		return payload
	}
	sectionMeta := make(map[string]DashboardTableMeta)
	for key, value := range payload {
		if key == "table_sections_meta" {
			continue
		}
		rows, ok := value.([]any)
		if !ok || len(rows) == 0 {
			continue
		}
		if _, ok := rows[0].(map[string]any); !ok {
			continue
		}
		total := len(rows)
		if total <= maxDashboardTableRows {
			continue
		}
		payload[key] = rows[:maxDashboardTableRows]
		sectionMeta[key] = DashboardTableMeta{Truncated: true, Total: total}
	}
	if len(sectionMeta) > 0 {
		meta := make(map[string]any, len(sectionMeta))
		for key, item := range sectionMeta {
			meta[key] = item
		}
		payload["table_sections_meta"] = meta
	}
	return payload
}

func capDashboardPayload(v any) (map[string]any, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return capDashboardTableSections(payload), nil
}

func writeRoleDashboardJSON(w http.ResponseWriter, resp any) {
	payload, err := capDashboardPayload(resp)
	if err != nil {
		httpresponse.JSON(w, http.StatusOK, resp)
		return
	}
	httpresponse.JSON(w, http.StatusOK, payload)
}
