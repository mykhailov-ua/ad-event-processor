export const CAMPAIGN_LIST_FILTER_TOTALS_MAX = 5000;

export const CAMPAIGN_LIST_EXPORT_MAX_ROWS = 5000;

/** Matches POST /campaigns/bulk-action campaign_ids cap (server bulkCampaignMaxSync). */
export const CAMPAIGN_LIST_BULK_CHUNK_SIZE = 50;

/** Matches GET /campaigns/export ids cap (server CampaignExportBatchMaxIDs). */
export const CAMPAIGN_LIST_EXPORT_BATCH_CHUNK_SIZE = 50;
