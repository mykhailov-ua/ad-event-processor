package campaign

import (
	"encoding/json"

	"ad-event-processor/internal/flow"
	"ad-event-processor/internal/reports"
)

type DataFreshnessDTO = reports.DataFreshnessDTO

type FlowDTO = flow.DTO
type FlowPathDTO = flow.PathDTO
type FlowPathLanderRef = flow.PathLanderRef
type FlowPathOfferRef = flow.PathOfferRef
type FlowPathFiltersDTO = flow.PathFiltersDTO
type FlowPathErrorDTO = flow.PathErrorDTO
type FlowValidateResponseDTO = flow.ValidateResponseDTO

func ValidateFlowPathShape(paths []FlowPathDTO) error {
	return flow.ValidatePathShape(paths)
}

func BuildCampaignFlowValidateResponse(paths []FlowPathDTO) FlowValidateResponseDTO {
	return flow.BuildValidateResponse(paths)
}

func ParseFlowPaths(raw json.RawMessage) ([]FlowPathDTO, error) {
	return flow.ParsePaths(raw)
}

func FormatFlowPathErrors(pathErrors []FlowPathErrorDTO) string {
	return flow.FormatPathErrors(pathErrors)
}
