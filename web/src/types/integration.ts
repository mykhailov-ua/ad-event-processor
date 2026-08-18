export type IntegrationSchemaDTO = {
  id: string;
  name: string;
  version: number;
  kind: string;
  schema: unknown;
  created_at: string;
  updated_at: string;
};

export type CreateIntegrationSchemaBody = {
  name: string;
  version: number;
  schema: unknown;
};

export type IntegrationTemplateCatalogEntry = {
  name: string;
  file: string;
  version: number;
  category: string;
  kind: string;
};

export type ApplyCampaignTemplatesResult = {
  campaign_id: string;
  traffic_source?: Record<string, string>;
  affiliate_postback?: Record<string, string>;
  affiliate_status?: Record<string, string>;
};
