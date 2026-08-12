/**
 * Admin `/api/v1` DTO types — field names match Go `json` tags (§12.3).
 * Prefer these over ad-hoc `Record<string, unknown>` bags in views.
 */

export type { AuthUser, LoginResponse, MeResponse } from './auth.js';
export type {
  CampaignDTO,
  CampaignListResponse,
  CampaignPatchBody,
  CampaignMarginDTO,
  BuyerCampaignPortfolioRow,
  BuyerPortfolioResponse,
} from './campaign.js';
export type {
  DataFreshness,
  ReportEnvelope,
  ReportCompareDeltas,
  PlacementReportRow,
  KeywordReportRow,
  TrueRoiRow,
  ReportRow,
} from './report.js';
export type {
  DoctorCheck,
  OpsDoctorSummary,
  DashboardSummary,
  IncidentSnapshot,
  ShardHealthStatus,
  OutboxHealthSummary,
} from './ops.js';
export type {
  LedgerEntryDTO,
  LedgerListResponse,
  InvoiceDTO,
  InvoiceLineDTO,
  InvoiceListResponse,
  WalletBalanceDTO,
  BillingInvariantDTO,
  InvoiceDeliveryDTO,
  InvoiceDeliveryListResponse,
  BillingExportJobDTO,
  BillingExportCreateSpec,
} from './billing.js';
export type {
  CustomerDTO,
  CustomerListResponse,
  TaxProfileDTO,
} from './customer.js';
export type {
  BlacklistEntryDTO,
  BlacklistListResponse,
  OutboxEventDTO,
  OutboxListResponse,
  AuditLogRow,
  DLQEntryDTO,
  DLQListResponse,
  FanOutSourceError,
} from './ops_extra.js';
export type {
  RtbDealDTO,
  RtbDealCreateSpec,
  RtbDealUpdateSpec,
  RtbFloorSuggestionDTO,
  RtbFloorsApplyResult,
} from './rtb.js';
export type { LicenseStatusDTO } from './license.js';
