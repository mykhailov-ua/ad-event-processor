/** Column ids and export defaults for campaigns directory CSV (select-table UI has no metrics grid). */

export const CAMPAIGN_LIST_PINNED_DATA_COLUMNS: CampaignListReorderableColumnId[] = ['name'];

export const CAMPAIGN_LIST_PRIORITY_METRICS: CampaignListMiddleColumnId[] = [
  'roi',
  'profit',
  'revenue',
  'cost',
  'clicks',
  'ctr',
  'cr',
  'approved',
  'approve_rate',
  'leads',
  'budget_pct',
  'epc',
  'cpc',
];

export type CampaignListMiddleColumnId =
  | 'status'
  | 'clicks'
  | 'impressions'
  | 'ctr'
  | 'unique_clicks'
  | 'lp_clicks'
  | 'lp_views'
  | 'group'
  | 'lp_ctr'
  | 'cr'
  | 'leads'
  | 'approved'
  | 'hold_leads'
  | 'rejected_leads'
  | 'approve_rate'
  | 'epc'
  | 'cpc'
  | 'cpa'
  | 'ecpa'
  | 'cpm'
  | 'blocks'
  | 'block_pct'
  | 'bots'
  | 'bot_pct'
  | 'revenue'
  | 'cost'
  | 'profit'
  | 'roi'
  | 'budget_pct'
  | 'flow'
  | 'owner'
  | 'countries';

export type CampaignListColumnId = 'select' | 'id' | 'name' | CampaignListMiddleColumnId;

export type CampaignListDataColumnId = Exclude<CampaignListColumnId, 'select'>;

export type CampaignListReorderableColumnId = 'name' | CampaignListMiddleColumnId;

export const CAMPAIGN_LIST_MIDDLE_COLUMNS: CampaignListMiddleColumnId[] = [
  'status',
  ...CAMPAIGN_LIST_PRIORITY_METRICS,
  'cpa',
  'ecpa',
  'cpm',
  'impressions',
  'unique_clicks',
  'lp_clicks',
  'lp_views',
  'lp_ctr',
  'hold_leads',
  'rejected_leads',
  'blocks',
  'block_pct',
  'bots',
  'bot_pct',
  'group',
  'flow',
  'owner',
  'countries',
];

export const CAMPAIGN_LIST_DEFAULT_HIDDEN: CampaignListMiddleColumnId[] = [
  'impressions',
  'unique_clicks',
  'rejected_leads',
  'bots',
  'cpa',
];

export type CampaignListColumnPrefs = {
  dataColumnOrder: CampaignListReorderableColumnId[];
  hidden: CampaignListMiddleColumnId[];
};

export const CAMPAIGN_LIST_COLUMN_LABELS: Record<CampaignListColumnId, string> = {
  select: '',
  id: 'ID',
  name: 'Name',
  status: 'Status',
  clicks: 'Clicks',
  impressions: 'Impressions',
  ctr: 'CTR',
  unique_clicks: 'Unique clicks',
  lp_clicks: 'LP clicks',
  lp_views: 'LP views',
  group: 'Group',
  lp_ctr: 'LP CTR',
  cr: 'CR',
  leads: 'Leads (raw)',
  approved: 'Approved',
  hold_leads: 'Hold',
  rejected_leads: 'Rejected',
  approve_rate: 'AR %',
  epc: 'EPC',
  cpc: 'CPC',
  cpa: 'CPA (raw)',
  ecpa: 'eCPA',
  cpm: 'CPM',
  blocks: 'Blocks',
  block_pct: 'Block %',
  bots: 'Bots',
  bot_pct: 'Bot %',
  revenue: 'Revenue',
  cost: 'Cost',
  profit: 'Profit',
  roi: 'ROI',
  budget_pct: 'Budget used',
  flow: 'Flow',
  owner: 'Owner',
  countries: 'Countries',
};

/** Shared min widths for ghost dashboard tables that reuse campaign column tokens. */
export const CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX: Record<CampaignListColumnId, number> = {
  select: 48,
  id: 72,
  name: 200,
  status: 148,
  clicks: 60,
  impressions: 72,
  ctr: 56,
  unique_clicks: 72,
  lp_clicks: 64,
  lp_views: 64,
  group: 112,
  lp_ctr: 64,
  cr: 48,
  leads: 64,
  approved: 64,
  hold_leads: 56,
  rejected_leads: 64,
  approve_rate: 56,
  epc: 52,
  cpc: 52,
  cpa: 64,
  ecpa: 56,
  cpm: 56,
  blocks: 56,
  block_pct: 56,
  bots: 48,
  bot_pct: 56,
  revenue: 80,
  cost: 72,
  profit: 72,
  roi: 56,
  budget_pct: 72,
  flow: 88,
  owner: 120,
  countries: 104,
};

const MIDDLE_COLUMN_SET = new Set<CampaignListMiddleColumnId>(CAMPAIGN_LIST_MIDDLE_COLUMNS);

function migrateLegacyColumnId(raw: string): CampaignListReorderableColumnId | null {
  if (raw === 'h_leads') {
    return 'hold_leads';
  }
  if (raw === 'tags') {
    return null;
  }
  if (raw === 'name') {
    return 'name';
  }
  if (MIDDLE_COLUMN_SET.has(raw as CampaignListMiddleColumnId)) {
    return raw as CampaignListMiddleColumnId;
  }
  return null;
}

export function isCampaignListMiddleColumnId(
  id: CampaignListColumnId
): id is CampaignListMiddleColumnId {
  return MIDDLE_COLUMN_SET.has(id as CampaignListMiddleColumnId);
}

export function defaultCampaignListColumnPrefs(): CampaignListColumnPrefs {
  return {
    dataColumnOrder: ['name', ...CAMPAIGN_LIST_MIDDLE_COLUMNS],
    hidden: [...CAMPAIGN_LIST_DEFAULT_HIDDEN],
  };
}

function normalizeDataColumnOrder(order: ReadonlyArray<string>): CampaignListReorderableColumnId[] {
  const seen = new Set<CampaignListReorderableColumnId>();
  const result: CampaignListReorderableColumnId[] = [];

  for (const raw of order) {
    const columnId = migrateLegacyColumnId(raw);
    if (!columnId || seen.has(columnId)) {
      continue;
    }
    seen.add(columnId);
    result.push(columnId);
  }

  if (!seen.has('name')) {
    result.unshift('name');
    seen.add('name');
  }

  for (const columnId of CAMPAIGN_LIST_MIDDLE_COLUMNS) {
    if (!seen.has(columnId)) {
      result.push(columnId);
    }
  }

  return result;
}

function normalizeHidden(hidden: ReadonlyArray<string>): CampaignListMiddleColumnId[] {
  const set = new Set<CampaignListMiddleColumnId>();
  for (const raw of hidden) {
    const columnId = migrateLegacyColumnId(raw);
    if (!columnId || columnId === 'name') {
      continue;
    }
    set.add(columnId);
  }
  return CAMPAIGN_LIST_MIDDLE_COLUMNS.filter((id) => set.has(id));
}

export function visibleCampaignListColumns(prefs: CampaignListColumnPrefs): CampaignListColumnId[] {
  const hidden = new Set(prefs.hidden);
  const ordered = normalizeDataColumnOrder(prefs.dataColumnOrder);
  const pinned: CampaignListColumnId[] = ['select', 'id', 'name'];
  const rest = ordered.filter((columnId) => columnId !== 'name' && !hidden.has(columnId));
  return [...pinned, ...rest];
}

export function defaultCampaignListExportDataColumns(): CampaignListDataColumnId[] {
  return visibleCampaignListColumns(defaultCampaignListColumnPrefs()).filter(
    (columnId): columnId is CampaignListDataColumnId => columnId !== 'select'
  );
}
