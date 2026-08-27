package migrationsource

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type binomReportRow struct {
	ID     json.RawMessage `json:"id"`
	Name   string          `json:"name"`
	TSName string          `json:"ts_name"`
	URL    string          `json:"url"`
	Cost   string          `json:"cost"`
	Budget float64         `json:"budget"`
}

type binomReportExport struct {
	Rows []binomReportRow `json:"rows"`
}

func binomReportIDString(raw json.RawMessage) string {
	raw = bytesTrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	var n int64
	if err := json.Unmarshal(raw, &n); err == nil {
		return strconv.FormatInt(n, 10)
	}
	return strings.TrimSpace(string(raw))
}

func ParseBinomReportAPI(payload []byte) (NormalizedBundle, error) {
	var rows []binomReportRow
	err := decodeJSONArrayOrCampaignsWrapper(payload,
		func(b []byte) error {
			return json.Unmarshal(b, &rows)
		},
		func(b []byte) error {
			var doc binomReportExport
			if err := json.Unmarshal(b, &doc); err != nil {
				return fmt.Errorf("decode binom report export: %w", err)
			}
			rows = doc.Rows
			return nil
		},
	)
	if err != nil {
		return NormalizedBundle{}, err
	}
	if len(rows) == 0 {
		return NormalizedBundle{}, fmt.Errorf("no campaigns in payload")
	}
	out := NormalizedBundle{SourceKind: SourceKindBinomReportAPI}
	for i, row := range rows {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			return NormalizedBundle{}, fmt.Errorf("campaign index %d missing name", i)
		}
		trackingURL := strings.TrimSpace(row.URL)
		if trackingURL == "" {
			return NormalizedBundle{}, fmt.Errorf("campaign index %d missing url (Binom report wire)", i)
		}
		id := binomReportIDString(row.ID)
		ref := "binom:" + id
		if id == "" {
			ref = fmt.Sprintf("binom:%d", i+1)
		}
		out.Campaigns = append(out.Campaigns, NormalizedCampaign{
			Ref:               ref,
			Name:              name,
			TrafficSourceName: strings.TrimSpace(row.TSName),
			TrackingURL:       trackingURL,
			BudgetUSD:         row.Budget,
		})
	}
	return out, nil
}
