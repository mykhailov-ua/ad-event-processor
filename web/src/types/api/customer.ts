/** Customer list/detail DTOs — mirror Go adminapi.CustomerDTO / TaxProfileDTO. */

export type CustomerDTO = {
  id: string;
  name?: string;
  balance?: number | string;
  currency?: string;
  active_campaigns?: number;
  total_spend?: number | string;
  created_at?: string;
  updated_at?: string;
  [key: string]: unknown;
};

export type CustomerListResponse = {
  items?: CustomerDTO[];
  total?: number;
};

export type TaxProfileDTO = {
  customer_id?: string;
  country_code?: string;
  tax_region?: string;
  tax_scheme?: string;
  tax_rate_bps?: number;
  [key: string]: unknown;
};
