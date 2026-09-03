import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { CampaignStatusTotals } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin, SelfServeCampaignTemplate } from '@/api/types';
import type { CustomerComboboxOption } from '@/shell/customer_combobox';
import type { CampaignsListFilterOption } from '@/domains/campaigns/list/campaigns_list_filter_select';
import type { CampaignListFilterTotalsView } from '@/domains/campaigns/list/campaign_list_filter_totals';
import type { CampaignListFilterQuery } from '@/domains/campaigns/list/campaigns_list_query';
import type {
  CampaignPacingFilter,
  CampaignSortField,
  CampaignStatusFilter,
  SortOrder,
} from '@/domains/campaigns/list/campaigns_list_types';

export type {
  CampaignPacingFilter,
  CampaignSortField,
  CampaignStatusFilter,
  SortOrder,
} from '@/domains/campaigns/list/campaigns_list_types';

export type CampaignListColumnWidthProbe = {
  items: Campaign[];
  metricsById: Record<string, CampaignListMetrics>;
  marginsById: Record<string, CampaignMargin>;
};

export type CampaignsDirectoryProps = {
  items: Campaign[];
  total: number;
  limit: number;
  offset: number;
  statusTotals: CampaignStatusTotals | undefined;
  statusTotalsLoading: boolean;
  customerOptions: CustomerComboboxOption[];
  customersLoading: boolean;
  customerNameById: Record<string, string>;
  metricsById: Record<string, CampaignListMetrics>;
  marginsById: Record<string, CampaignMargin>;
  columnWidthProbe?: CampaignListColumnWidthProbe;
  appliedCustomerId: string;
  appliedStatus: CampaignStatusFilter;
  appliedSort: CampaignSortField;
  appliedOrder: SortOrder;
  draftCustomerId: string;
  draftStatus: CampaignStatusFilter;
  draftPacing: CampaignPacingFilter;
  draftOwnerUserId: string;
  draftCountry: string;
  draftBudgetMinUsd: string;
  draftBudgetMaxUsd: string;
  draftStatsFrom: string;
  draftStatsTo: string;
  ownerOptions: CampaignsListFilterOption[];
  ownerEmailById: Record<string, string>;
  countryOptions: CampaignsListFilterOption[];
  listFacetsFetching?: boolean;
  filterTotals?: CampaignListFilterTotalsView;
  exportFilterQuery: CampaignListFilterQuery;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  filtersActive: boolean;
  customerId: string | undefined;
  createSectionOpen: boolean;
  onCreateSectionOpenChange: (open: boolean) => void;
  templates: SelfServeCampaignTemplate[];
  templatesLoading: boolean;
  templatesError: Error | undefined;
  draftTemplateId: string;
  draftCreateName: string;
  draftBudgetLimitMicro: string;
  creating: boolean;
  actionError: Error | undefined;
  onDraftCustomerIdChange: (customerId: string) => void;
  onDraftStatusChange: (status: CampaignStatusFilter) => void;
  onDraftPacingChange: (pacing: CampaignPacingFilter) => void;
  onDraftOwnerUserIdChange: (ownerUserId: string) => void;
  onDraftCountryChange: (country: string) => void;
  onDraftBudgetMinUsdChange: (value: string) => void;
  onDraftBudgetMaxUsdChange: (value: string) => void;
  onBudgetFiltersApply: () => void;
  onStatsRangeChange: (from: string, to: string) => void;
  onRefreshList: () => void;
  onColumnSort: (field: CampaignSortField) => void;
  onPageChange: (nextOffset: number) => void;
  onPageSizeChange: (size: number) => void;
  onDraftTemplateIdChange: (templateId: string) => void;
  onDraftCreateNameChange: (name: string) => void;
  onDraftBudgetLimitMicroChange: (value: string) => void;
  onLoadTemplates: () => void;
  onCreateCampaign: () => void;
};
