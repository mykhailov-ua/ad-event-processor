import type { CampaignListMiddleColumnId } from '@/domains/campaigns/list/campaign_list_columns';

export type CampaignSortField =
  | CampaignListMiddleColumnId
  | 'id'
  | 'name'
  | 'updated_at'
  | 'spend'
  | 'budget_limit';

export type SortOrder = 'asc' | 'desc';

export type CampaignStatusFilter = '' | 'ACTIVE' | 'PAUSED' | 'ARCHIVED';

export type CampaignPacingFilter = '' | 'EVEN' | 'ASAP';
