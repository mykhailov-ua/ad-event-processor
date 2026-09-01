export type { components, operations } from './generated/openapi.js';

import type { components } from './generated/openapi.js';

export type Customer = components['schemas']['Customer'];
export type CustomerListResponse = components['schemas']['CustomerListResponse'];
export type ListSort = components['schemas']['ListSort'];
export type SessionResponse = components['schemas']['SessionResponse'];
export type SessionNavItem = components['schemas']['SessionNavItem'];
export type ErrorBody = components['schemas']['ErrorBody'];

export type Campaign = components['schemas']['Campaign'];
export type CampaignListResponse = components['schemas']['CampaignListResponse'];
export type PatchCampaignRequest = components['schemas']['PatchCampaignRequest'];
export type BillingSummary = components['schemas']['BillingSummary'];
export type Invoice = components['schemas']['Invoice'];
export type InvoiceListResponse = components['schemas']['InvoiceListResponse'];
export type BillingStatement = components['schemas']['BillingStatement'];
export type BillingForecast = components['schemas']['BillingForecast'];
export type Wallet = components['schemas']['Wallet'];
export type PaymentHistoryListResponse = components['schemas']['PaymentHistoryListResponse'];
export type PaymentSummary = components['schemas']['PaymentSummary'];
export type PaymentHistoryRow = components['schemas']['PaymentHistoryRow'];
export type BillingInvoiceLine = components['schemas']['BillingInvoiceLine'];
export type TaxProfile = components['schemas']['TaxProfile'];
export type SelfServeCampaignTemplate = components['schemas']['SelfServeCampaignTemplate'];
export type SelfServeTemplateListResponse =
  components['schemas']['SelfServeTemplateListResponse'];
export type SelfServeCreateCampaignRequest =
  components['schemas']['SelfServeCreateCampaignRequest'];
