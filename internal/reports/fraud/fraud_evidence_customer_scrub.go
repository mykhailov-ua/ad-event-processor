package fraud

import "ad-event-processor/internal/reports"

func ScrubCustomerFraudEvidencePack(pack reports.FraudEvidencePackDTO) reports.FraudEvidencePackDTO {
	out := pack
	out.Timeline = scrubFraudEvidenceTimelineRows(out.Timeline)
	out.FraudEvents = scrubFraudEvidenceFraudRows(out.FraudEvents)
	out.Signals = aggregateFraudEvidenceSignals(out.FraudEvents)
	return out
}

func scrubFraudEvidenceTimelineRows(rows []reports.FraudEvidenceTimelineRowDTO) []reports.FraudEvidenceTimelineRowDTO {
	if len(rows) == 0 {
		return rows
	}
	out := make([]reports.FraudEvidenceTimelineRowDTO, len(rows))
	for i := range rows {
		out[i] = rows[i]
		out[i].Country = ""
		out[i].Sub1 = ""
		out[i].PlacementID = ""
	}
	return out
}

func scrubFraudEvidenceFraudRows(rows []reports.FraudEvidenceFraudRowDTO) []reports.FraudEvidenceFraudRowDTO {
	if len(rows) == 0 {
		return rows
	}
	out := make([]reports.FraudEvidenceFraudRowDTO, len(rows))
	for i := range rows {
		row := rows[i]
		category, label := FraudReasonToCategory(row.FraudReason)
		if label != "" {
			row.FraudReason = label
		} else {
			row.FraudReason = FraudCategoryLabel(category)
		}
		row.PlacementID = ""
		out[i] = row
	}
	return out
}

var customerFraudEvidencePerms = []string{"campaigns:read"}

func reportPermsCustomerFraudEvidence() []string {
	return customerFraudEvidencePerms
}
