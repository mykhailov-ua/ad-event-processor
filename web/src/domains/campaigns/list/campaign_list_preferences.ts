import type { CampaignListMiddleColumnId } from '@/domains/campaigns/list/campaign_list_columns';
import {
  CAMPAIGN_LIST_COLUMN_LABELS,
  CAMPAIGN_LIST_DEFAULT_HIDDEN,
  CAMPAIGN_LIST_MIDDLE_COLUMNS,
  defaultCampaignListColumnPrefs,
  type CampaignListColumnPrefs,
} from '@/domains/campaigns/list/campaign_list_columns';

export type CampaignListColumnPresetId = 'full' | 'traffic' | 'finance' | 'minimal';

export const CAMPAIGN_LIST_COLUMN_PRESET_LABELS: Record<CampaignListColumnPresetId, string> = {
  full: 'Full',
  traffic: 'Traffic',
  finance: 'Finance',
  minimal: 'Minimal',
};

const TRAFFIC_COLUMNS: CampaignListMiddleColumnId[] = [
  'status',
  'clicks',
  'impressions',
  'ctr',
  'lp_clicks',
  'lp_views',
  'lp_ctr',
  'cr',
  'leads',
  'approved',
  'hold_leads',
  'approve_rate',
  'blocks',
  'block_pct',
  'bots',
  'bot_pct',
  'group',
  'countries',
];

const FINANCE_COLUMNS: CampaignListMiddleColumnId[] = [
  'status',
  'clicks',
  'leads',
  'approved',
  'approve_rate',
  'revenue',
  'cost',
  'profit',
  'roi',
  'cpc',
  'cpa',
  'ecpa',
  'cpm',
  'epc',
  'budget_pct',
];

const MINIMAL_COLUMNS: CampaignListMiddleColumnId[] = [
  'status',
  'clicks',
  'approved',
  'approve_rate',
  'profit',
  'roi',
];

export type CampaignListColumnCategoryId = 'campaign' | 'traffic' | 'funnel' | 'finance';

export type CampaignListColumnCategory = {
  id: CampaignListColumnCategoryId;
  title: string;
  columns: CampaignListMiddleColumnId[];
};

export const CAMPAIGN_LIST_COLUMN_CATEGORIES: CampaignListColumnCategory[] = [
  {
    id: 'campaign',
    title: 'Campaign',
    columns: ['status', 'group', 'flow', 'owner', 'countries', 'tags', 'budget_pct'],
  },
  {
    id: 'traffic',
    title: 'Traffic',
    columns: [
      'clicks',
      'impressions',
      'ctr',
      'unique_clicks',
      'lp_clicks',
      'lp_views',
      'lp_ctr',
      'cr',
      'blocks',
      'block_pct',
      'bots',
      'bot_pct',
    ],
  },
  {
    id: 'funnel',
    title: 'Funnel',
    columns: ['leads', 'approved', 'hold_leads', 'rejected_leads', 'approve_rate'],
  },
  {
    id: 'finance',
    title: 'Finance',
    columns: ['revenue', 'cost', 'profit', 'roi', 'epc', 'cpc', 'cpa', 'ecpa', 'cpm'],
  },
];

export function campaignListColumnOptions(): { id: CampaignListMiddleColumnId; label: string }[] {
  return CAMPAIGN_LIST_MIDDLE_COLUMNS.map((id) => ({
    id,
    label: CAMPAIGN_LIST_COLUMN_LABELS[id],
  }));
}

export function campaignListColumnPrefsFromPreset(
  presetId: CampaignListColumnPresetId,
): CampaignListColumnPrefs {
  const visible =
    presetId === 'traffic'
      ? TRAFFIC_COLUMNS
      : presetId === 'finance'
        ? FINANCE_COLUMNS
        : presetId === 'minimal'
          ? MINIMAL_COLUMNS
          : CAMPAIGN_LIST_MIDDLE_COLUMNS.filter(
              (id) => !CAMPAIGN_LIST_DEFAULT_HIDDEN.includes(id),
            );

  const visibleSet = new Set(visible);
  return {
    dataColumnOrder: ['name', ...CAMPAIGN_LIST_MIDDLE_COLUMNS],
    hidden: CAMPAIGN_LIST_MIDDLE_COLUMNS.filter((id) => !visibleSet.has(id)),
    widthPx: {},
  };
}

export function defaultCampaignListPreferencesPrefs(): CampaignListColumnPrefs {
  return defaultCampaignListColumnPrefs();
}
