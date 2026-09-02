import type { CampaignListMiddleColumnId } from '@/domains/campaigns/campaign_list_columns';

export type CampaignServerSortField = 'name' | 'updated_at' | 'spend' | 'budget_limit';

export type CampaignSortField = CampaignServerSortField | CampaignListMiddleColumnId | 'id';

export type SortOrder = 'asc' | 'desc';

export type CampaignStatusFilter = '' | 'ACTIVE' | 'PAUSED' | 'ARCHIVED';
