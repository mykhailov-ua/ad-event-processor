package controlplane

import (
	"ad-event-processor/internal/campaign"
)

type (
	CampaignDTO                           = campaign.CampaignDTO
	CampaignListResponse                  = campaign.CampaignListResponse
	CampaignEventListResponse             = campaign.CampaignEventListResponse
	CampaignEventDTO                      = campaign.CampaignEventDTO
	PatchCampaignRequest                  = campaign.PatchCampaignRequest
	CloneCampaignSpec                     = campaign.CloneCampaignSpec
	CloneCampaignResult                   = campaign.CloneCampaignResult
	CampaignPublishCheckDTO               = campaign.CampaignPublishCheckDTO
	CampaignSmokeResultDTO                = campaign.CampaignSmokeResultDTO
	ConversionMappingDTO                  = campaign.ConversionMappingDTO
	CampaignWizardSessionDTO              = campaign.CampaignWizardSessionDTO
	CampaignWizardCommitResult            = campaign.CampaignWizardCommitResult
	CampaignWizardStored                  = campaign.CampaignWizardStored
	IntegrationHealthDTO                  = campaign.IntegrationHealthDTO
	IntegrationHealthRow                  = campaign.IntegrationHealthRow
	IntegrationHealthInput                = campaign.IntegrationHealthInput
	CampaignsHTTPHandlers                 = campaign.CampaignsHTTPHandlers
	CampaignReader                        = campaign.CampaignReader
	CampaignRuntime                       = campaign.Runtime
	CampaignFlowPathValidator             = campaign.CampaignFlowPathValidator
	ConversionMappingService              = campaign.ConversionMappingService
	CampaignMarginDTO                     = campaign.CampaignMarginDTO
	CampaignExportBundle                  = campaign.CampaignExportBundle
	CampaignExportCampaign                = campaign.CampaignExportCampaign
	CampaignExportLander                  = campaign.CampaignExportLander
	CampaignExportOffer                   = campaign.CampaignExportOffer
	CampaignExportFlow                    = campaign.CampaignExportFlow
	CampaignExportFlowPath                = campaign.CampaignExportFlowPath
	CampaignExportFlowLanderRef           = campaign.CampaignExportFlowLanderRef
	CampaignExportFlowOfferRef            = campaign.CampaignExportFlowOfferRef
	CampaignExportPostback                = campaign.CampaignExportPostback
	ImportCampaignSpec                    = campaign.ImportCampaignSpec
	ImportCampaignResult                  = campaign.ImportCampaignResult
	ImportMigrationSpec                   = campaign.ImportMigrationSpec
	ImportMigrationFailure                = campaign.ImportMigrationFailure
	ImportMigrationResult                 = campaign.ImportMigrationResult
	ImportValidateJobRequest              = campaign.ImportValidateJobRequest
	PullMigrationPreviewSpec              = campaign.PullMigrationPreviewSpec
	PullMigrationImportSpec               = campaign.PullMigrationImportSpec
	CampaignFraudConfigDTO                = campaign.CampaignFraudConfigDTO
	PatchCampaignFraudRequest             = campaign.PatchCampaignFraudRequest
	CampaignFraudPreviewDTO               = campaign.CampaignFraudPreviewDTO
	FraudPreviewTierCountsDTO             = campaign.FraudPreviewTierCountsDTO
	PreviewCampaignFraudRequest           = campaign.PreviewCampaignFraudRequest
	CampaignFraudService                  = campaign.CampaignFraudService
	CustomerBalanceDTO                    = campaign.CustomerBalanceDTO
	BalanceLedgerDTO                      = campaign.BalanceLedgerDTO
	LedgerExportResult                    = campaign.LedgerExportResult
	UsageExportResult                     = campaign.UsageExportResult
	FlowPathFiltersDTO                    = campaign.FlowPathFiltersDTO
	FlowPathErrorDTO                      = campaign.FlowPathErrorDTO
	CloneCampaignOptions                  = campaign.CloneCampaignOptions
	IngressCostConfigDTO                  = campaign.IngressCostConfigDTO
	AuditLogDTO                           = campaign.AuditLogDTO
	MutationPreviewDTO                    = campaign.MutationPreviewDTO
	BlacklistDTO                          = campaign.BlacklistDTO
	CampaignSmokeRedirectHop              = campaign.CampaignSmokeRedirectHop
	CampaignWizardReviewDTO               = campaign.CampaignWizardReviewDTO
	CampaignWizardTrafficSourceStep       = campaign.CampaignWizardTrafficSourceStep
	CampaignWizardIntegrationTemplateStep = campaign.CampaignWizardIntegrationTemplateStep
	CampaignWizardFlowSkeletonStep        = campaign.CampaignWizardFlowSkeletonStep
	CampaignWizardBudgetStep              = campaign.CampaignWizardBudgetStep
	CampaignWizardStepsDTO                = campaign.CampaignWizardStepsDTO
	CampaignWizardPreviewDTO              = campaign.CampaignWizardPreviewDTO
	CampaignWizardAssetRef                = campaign.CampaignWizardAssetRef
	FlowPathDTO                           = campaign.FlowPathDTO
	FlowPathLanderRef                     = campaign.FlowPathLanderRef
	FlowPathOfferRef                      = campaign.FlowPathOfferRef
	CampaignStatsDTO                      = campaign.CampaignStatsDTO
	StatusHistoryDTO                      = campaign.StatusHistoryDTO
	CampaignHourlyBucketDTO               = campaign.CampaignHourlyBucketDTO
	CampaignDailyBucketDTO                = campaign.CampaignDailyBucketDTO
	SpendCurvePoint                       = campaign.SpendCurvePoint
	ForecastAdvisory                      = campaign.ForecastAdvisory
	CampaignForecastInput                 = campaign.CampaignForecastInput
	CampaignForecastDTO                   = campaign.CampaignForecastDTO
	FlowValidateResponseDTO               = campaign.FlowValidateResponseDTO
	BlacklistListResponse                 = campaign.BlacklistListResponse
	CampaignMetricsDTO                    = campaign.CampaignMetricsDTO
)

var (
	ErrForecastClickHouseTimeout = campaign.ErrForecastClickHouseTimeout
	ErrForecastUnavailable       = campaign.ErrForecastUnavailable
)
