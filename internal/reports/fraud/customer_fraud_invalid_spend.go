package fraud

import "ad-event-processor/internal/reports"

func AttachInvalidSpendKPI(out *reports.CustomerFraudOverviewDTO, blockedEvents, silentRejectEvents, totalEvents int64, spendMicros int64, attributionCoverage float64) {
	if out == nil {
		return
	}
	invalidEvents := blockedEvents + silentRejectEvents
	if totalEvents == 0 || spendMicros <= 0 {
		out.InvalidSpendMicros = 0
		out.InvalidSpendDisplay = reports.FormatMicro(0)
		out.InvalidSpendSharePct = 0
		out.ShareLabel = "0.0% of spend"
		if attributionCoverage > 0 && attributionCoverage < 0.9 {
			out.Disclaimer = invalidSpendDisclaimer(attributionCoverage)
		}
		return
	}
	share := float64(invalidEvents) / float64(totalEvents)
	invalidMicros := int64(float64(spendMicros) * share)
	out.InvalidSpendMicros = invalidMicros
	out.InvalidSpendDisplay = reports.FormatMicro(invalidMicros)
	out.InvalidSpendSharePct = share
	out.ShareLabel = reports.FormatRateDisplay(share) + " of spend"
	if attributionCoverage > 0 && attributionCoverage < 0.9 {
		out.Disclaimer = invalidSpendDisclaimer(attributionCoverage)
	}
}

func invalidSpendDisclaimer(coverage float64) string {
	return "Spend attribution coverage below 90%; invalid spend estimate may be incomplete."
}

func ComputeAttributionCoverage(totalEvents, attributedEvents int64) float64 {
	if totalEvents == 0 {
		return 0
	}
	return float64(attributedEvents) / float64(totalEvents)
}
