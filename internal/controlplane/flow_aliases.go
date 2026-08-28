package controlplane

import "ad-event-processor/internal/flow"

type (
	FlowHTTPHandlers           = flow.HTTPHandlers
	FlowService                = flow.Service
	LanderDTO                  = flow.LanderDTO
	OfferDTO                   = flow.OfferDTO
	CreateLanderRequest        = flow.CreateLanderRequest
	CreateOfferRequest         = flow.CreateOfferRequest
	CreateFlowRequest          = flow.CreateFlowRequest
	UpdateFlowRequest          = flow.UpdateFlowRequest
	FlowDTO                    = flow.DTO
	HostedEditorFileDTO        = flow.HostedEditorFileDTO
	HostedEditorStateDTO       = flow.HostedEditorStateDTO
	HostedEditorFileBodyDTO    = flow.HostedEditorFileBodyDTO
	HostedEditorSaveResultDTO  = flow.HostedEditorSaveResultDTO
	HostedEditorPublishRequest = flow.HostedEditorPublishRequest
)

func validateFlowPathShape(paths []FlowPathDTO) error {
	return flow.ValidatePathShape(paths)
}
