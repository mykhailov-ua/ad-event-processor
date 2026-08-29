package campaign

import "strings"

func buildCampaignConflictResponse(current CampaignDTO, req PatchCampaignRequest) CampaignConflictResponseDTO {
	fields := conflictFieldsForPatch(current, req)
	return CampaignConflictResponseDTO{
		Error:          "campaign_revision_conflict",
		ServerRevision: campaignRevision(current.UpdatedAt),
		ConflictFields: fields,
		MergeHintLabel: "Reload the campaign and merge your changes",
		Current:        current,
	}
}

func conflictFieldsForPatch(current CampaignDTO, req PatchCampaignRequest) []string {
	var fields []string
	if req.Name != nil && *req.Name != current.Name {
		fields = append(fields, "name")
	}
	if req.Status != nil && !strings.EqualFold(*req.Status, current.Status) {
		fields = append(fields, "status")
	}
	if req.PacingMode != nil && !strings.EqualFold(*req.PacingMode, current.PacingMode) {
		fields = append(fields, "pacing_mode")
	}
	if req.BudgetLimit != nil && *req.BudgetLimit != current.BudgetLimit {
		fields = append(fields, "budget_limit")
	}
	if req.TargetURL != nil && *req.TargetURL != current.TargetURL {
		fields = append(fields, "target_url")
	}
	if len(fields) == 0 {
		fields = []string{"updated_at"}
	}
	return fields
}
