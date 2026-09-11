import type {
  Campaign,
  CampaignBulkAction,
  CampaignBulkActionRequest,
  CampaignBulkActionResponse,
  CampaignBulkActionResultRow,
  CampaignDiffResponse,
  CampaignDiffRow,
  CampaignExportBundle,
  CampaignFraudEditorSummary,
  CampaignGeoSummary,
  CampaignListMetricsRow,
  CampaignListMetricsTotalsResponse,
  CampaignPublishBlockedError,
  CampaignStatusTotals,
  CloneCampaignOptions,
  CloneCampaignPreview,
  CloneCampaignRequest,
  CloneCampaignResult,
  MacroPreviewRequest,
  MacroPreviewResponse,
} from './types.js';

export type {
  CampaignBulkAction,
  CampaignBulkActionRequest,
  CampaignBulkActionResponse,
  CampaignBulkActionResultRow,
  CampaignDiffResponse,
  CampaignDiffRow,
  CampaignFraudEditorSummary,
  CampaignGeoSummary,
  CloneCampaignOptions,
  CloneCampaignPreview,
  CloneCampaignRequest,
  CloneCampaignResult,
  MacroPreviewRequest,
  MacroPreviewResponse,
  CampaignStatusTotals,
  CampaignListMetricsTotalsResponse,
};

export type CampaignListFacetOwner = {
  user_id: string;
  email?: string;
};

export type CampaignListFacetsResponse = {
  countries: string[];
  owners: CampaignListFacetOwner[];
};

export type PublishCampaignResult =
  | { status: 'published'; campaign: Campaign }
  | { status: 'blocked'; error: CampaignPublishBlockedError };

export type CampaignListMetrics = Pick<
  CampaignListMetricsRow,
  | 'impressions'
  | 'clicks'
  | 'conversions'
  | 'unique_clicks'
  | 'blocks'
  | 'leads_raw'
  | 'hold_leads'
  | 'rejected_leads'
  | 'lp_clicks'
  | 'lp_views'
  | 'bots'
  | 'stale'
  | 'revenue_micro'
  | 'cost_micro'
  | 'profit_micro'
  | 'epc_micro'
  | 'cpc_micro'
  | 'cpa_micro'
  | 'ecpa_micro'
  | 'ctr_pct'
  | 'lp_ctr_pct'
  | 'cr_pct'
  | 'approve_rate_pct'
  | 'block_pct'
  | 'bot_pct'
  | 'roi_pct'
  | 'cpm_usd'
  | 'budget_burn_pct'
  | 'pacing_mode'
  | 'pacing_health'
  | 'metrics_stale'
  | 'metrics_as_of'
>;

export type CampaignExportBatchResponse = {
  items: Record<string, CampaignExportBundle>;
  errors?: { id: string; error_code?: string }[];
};

export const CAMPAIGN_BULK_ACTION_MAX_IDS = 50;

export type BulkCloneCampaignsRequest = {
  source_campaign_ids: string[];
  customer_id?: string;
  name_prefix?: string;
  name_suffix?: string;
  options?: CloneCampaignOptions;
};

export type BulkCloneCampaignResultRow = {
  source_id: string;
  id?: string;
  name?: string;
  ok: boolean;
  error_code?: string;
};

export type BulkCloneCampaignsResponse = {
  results: BulkCloneCampaignResultRow[];
};

export function summarizeCampaignBulkResults(results: CampaignBulkActionResultRow[]): {
  succeeded: CampaignBulkActionResultRow[];
  failed: CampaignBulkActionResultRow[];
} {
  const succeeded: CampaignBulkActionResultRow[] = [];
  const failed: CampaignBulkActionResultRow[] = [];
  for (const row of results) {
    if (row.ok) {
      succeeded.push(row);
    } else {
      failed.push(row);
    }
  }
  return { succeeded, failed };
}
