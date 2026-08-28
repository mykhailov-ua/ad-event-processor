package campaign

import (
	"net/http"
	"sort"

	"ad-event-processor/internal/integrationschema"
	"ad-event-processor/pkg/httpresponse"
)

func (h *IntegrationSchemaHTTPHandlers) listAffiliateStatusPresets(w http.ResponseWriter, r *http.Request) {
	out := make([]AffiliateStatusPresetDTO, 0, 8)
	for _, entry := range integrationschema.BundledIntegrationTemplateCatalog {
		if entry.Kind != integrationschema.KindStatusMapping {
			continue
		}
		_, kind, parsed, err := integrationschema.LoadBundledTemplate(entry)
		if err != nil || kind != integrationschema.KindStatusMapping {
			continue
		}
		statusSchema := parsed.(*integrationschema.StatusMappingSchema)
		statuses := make([]AffiliateStatusPresetEntryDTO, 0, len(statusSchema.StatusMap))
		for inbound, goal := range statusSchema.StatusMap {
			statuses = append(statuses, AffiliateStatusPresetEntryDTO{
				InboundStatus: inbound,
				GoalName:      goal,
			})
		}
		sort.Slice(statuses, func(i, j int) bool {
			return statuses[i].InboundStatus < statuses[j].InboundStatus
		})
		out = append(out, AffiliateStatusPresetDTO{
			Name:     entry.Name,
			Statuses: statuses,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	httpresponse.JSON(w, http.StatusOK, out)
}
