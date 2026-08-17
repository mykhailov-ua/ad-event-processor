/** GET /api/v1/integration/schemas — adminapi.IntegrationSchemaDTO */
export type IntegrationSchemaDTO = {
  id: string;
  name: string;
  version: number;
  kind: string;
  schema: unknown;
  created_at: string;
  updated_at: string;
};

/** POST /api/v1/integration/schemas */
export type CreateIntegrationSchemaBody = {
  name: string;
  version: number;
  schema: unknown;
};

/** GET /api/v1/integration/templates — bundled GM-M4 catalog entry */
export type IntegrationTemplateCatalogEntry = {
  name: string;
  file: string;
  version: number;
  category: string;
  kind: string;
};

/** POST /api/v1/campaigns/{id}/apply-templates result */
export type ApplyCampaignTemplatesResult = {
  campaign_id: string;
  traffic_source?: Record<string, string>;
  affiliate_postback?: Record<string, string>;
  affiliate_status?: Record<string, string>;
};
