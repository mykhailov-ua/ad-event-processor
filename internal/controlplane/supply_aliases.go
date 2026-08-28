package controlplane

import "ad-event-processor/internal/supply"

type SellerDTO = supply.SellerDTO
type AdsTxtEntryDTO = supply.AdsTxtEntryDTO
type SellerWriteRequest = supply.SellerWriteRequest
type AdsTxtWriteRequest = supply.AdsTxtWriteRequest
type SupplyExportPathDTO = supply.ExportPathDTO
type SupplyValidationDTO = supply.ValidationDTO
type SupplyChainNode = supply.ChainNode
type CampaignSupplyChainDTO = supply.CampaignChainDTO
type SellerCreateSpec = supply.SellerCreateSpec
type SellerUpdateSpec = supply.SellerUpdateSpec
type AdsTxtEntryCreateSpec = supply.AdsTxtEntryCreateSpec
type AdsTxtEntryUpdateSpec = supply.AdsTxtEntryUpdateSpec
type SupplyFilesPayload = supply.FilesPayload

var (
	ErrSellerNotFound      = supply.ErrSellerNotFound
	ErrAdsTxtEntryNotFound = supply.ErrAdsTxtEntryNotFound
	ErrInvalidSellerType   = supply.ErrInvalidSellerType
	ErrInvalidRelationship = supply.ErrInvalidRelationship
	ErrSupplyChainTooLong  = supply.ErrChainTooLong
	ErrSellersJSONInvalid  = supply.ErrSellersJSONInvalid
)

type SupplyHTTPHandlers = supply.HTTPHandlers
