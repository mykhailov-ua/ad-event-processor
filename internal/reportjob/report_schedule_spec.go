package reportjob

import (
	"encoding/json"
	"strings"
)

type reportScheduleSpec struct {
	From           string                   `json:"from"`
	To             string                   `json:"to"`
	FromOffsetDays int                      `json:"from_offset_days"`
	ToOffsetDays   int                      `json:"to_offset_days"`
	CompareFrom    string                   `json:"compare_from,omitempty"`
	CompareTo      string                   `json:"compare_to,omitempty"`
	RowLimit       int                      `json:"row_limit,omitempty"`
	GoogleSheet    ReportJobGoogleSheetSpec `json:"google_sheet,omitempty"`
	Notify         ReportJobNotifySpec      `json:"notify,omitempty"`
}

func parseReportScheduleSpec(specJSON []byte) reportScheduleSpec {
	if len(specJSON) == 0 {
		return reportScheduleSpec{}
	}
	var spec reportScheduleSpec
	_ = json.Unmarshal(specJSON, &spec)
	return spec
}

func mergeReportScheduleSpec(
	rangeSpec json.RawMessage,
	googleSheet ReportJobGoogleSheetSpec,
	notify ReportJobNotifySpec,
) (json.RawMessage, error) {
	base := make(map[string]json.RawMessage)
	if len(rangeSpec) > 0 {
		if err := json.Unmarshal(rangeSpec, &base); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(googleSheet.Mode) != "" ||
		strings.TrimSpace(googleSheet.SpreadsheetID) != "" ||
		strings.TrimSpace(googleSheet.SheetTitle) != "" {
		raw, err := json.Marshal(googleSheet)
		if err != nil {
			return nil, err
		}
		base["google_sheet"] = raw
	}
	if strings.TrimSpace(notify.Channel) != "" ||
		strings.TrimSpace(notify.Email) != "" ||
		strings.TrimSpace(notify.WebhookURL) != "" {
		raw, err := json.Marshal(notify)
		if err != nil {
			return nil, err
		}
		base["notify"] = raw
	}
	if len(base) == 0 {
		return json.RawMessage(`{}`), nil
	}
	return json.Marshal(base)
}

func populateReportScheduleDTOFromSpec(dto *ReportScheduleDTO, specJSON []byte) {
	if dto == nil {
		return
	}
	spec := parseReportScheduleSpec(specJSON)
	dto.GoogleSheet = spec.GoogleSheet
	dto.Notify = spec.Notify
}
