export const CAMPAIGN_LIST_COLUMNS_STORAGE_KEY = 'aed.campaigns.listColumns.v7';

export const COLUMN_DRAG_MIME = 'application/x-aed-campaign-column';

export type CampaignListMiddleColumnId =
  | 'status'
  | 'tags'
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

/** @deprecated Renamed to hold_leads; migrated on read from localStorage. */
export type LegacyCampaignListMiddleColumnId = CampaignListMiddleColumnId | 'h_leads';

export type CampaignListColumnId = 'select' | 'id' | 'name' | CampaignListMiddleColumnId;

export type CampaignListDataColumnId = Exclude<CampaignListColumnId, 'select'>;

export type CampaignListReorderableColumnId = 'name' | CampaignListMiddleColumnId;

export const CAMPAIGN_LIST_MIDDLE_COLUMNS: CampaignListMiddleColumnId[] = [
  'status',
  'clicks',
  'impressions',
  'ctr',
  'unique_clicks',
  'lp_clicks',
  'lp_views',
  'group',
  'lp_ctr',
  'leads',
  'approved',
  'hold_leads',
  'rejected_leads',
  'approve_rate',
  'cr',
  'blocks',
  'block_pct',
  'bots',
  'bot_pct',
  'epc',
  'cpc',
  'cpa',
  'ecpa',
  'cpm',
  'revenue',
  'cost',
  'profit',
  'roi',
  'budget_pct',
  'flow',
  'owner',
  'countries',
  'tags',
];

export const CAMPAIGN_LIST_DEFAULT_HIDDEN: CampaignListMiddleColumnId[] = [
  'tags',
  'impressions',
  'unique_clicks',
  'rejected_leads',
  'bots',
  'cpa',
];

export type CampaignListColumnPrefs = {
  dataColumnOrder: CampaignListReorderableColumnId[];
  hidden: CampaignListMiddleColumnId[];
  widthPx: Partial<Record<CampaignListColumnId, number>>;
};

export const CAMPAIGN_LIST_COLUMN_LABELS: Record<CampaignListColumnId, string> = {
  select: '',
  id: 'ID',
  name: 'Name',
  status: 'Status',
  tags: 'Tags',
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

export const CAMPAIGN_LIST_SELECTION_COLUMN_WIDTH_PX = 28;

export const CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX: Record<CampaignListColumnId, number> = {
  select: CAMPAIGN_LIST_SELECTION_COLUMN_WIDTH_PX,
  id: 72,
  name: 200,
  status: 88,
  tags: 56,
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
  countries: 72,
};

const CAMPAIGN_LIST_COLUMN_MAX_WIDTH_PX: Partial<Record<CampaignListColumnId, number>> = {
  select: CAMPAIGN_LIST_SELECTION_COLUMN_WIDTH_PX,
  id: 96,
  name: 320,
  status: 120,
  tags: 120,
  group: 220,
  owner: 220,
  flow: 160,
  countries: 160,
};

const CAMPAIGN_LIST_COLUMN_DEFAULT_MAX_WIDTH_PX = 160;

export function clampCampaignListColumnWidthPx(
  columnId: CampaignListColumnId,
  widthPx: number,
): number {
  const minWidth = CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[columnId];
  const maxWidth =
    CAMPAIGN_LIST_COLUMN_MAX_WIDTH_PX[columnId] ?? CAMPAIGN_LIST_COLUMN_DEFAULT_MAX_WIDTH_PX;
  return Math.min(maxWidth, Math.max(minWidth, Math.trunc(widthPx)));
}

const MIDDLE_COLUMN_SET = new Set<CampaignListMiddleColumnId>(CAMPAIGN_LIST_MIDDLE_COLUMNS);

function migrateLegacyColumnId(raw: string): CampaignListReorderableColumnId | null {
  if (raw === 'h_leads') {
    return 'hold_leads';
  }
  if (raw === 'name') {
    return 'name';
  }
  if (MIDDLE_COLUMN_SET.has(raw as CampaignListMiddleColumnId)) {
    return raw as CampaignListMiddleColumnId;
  }
  return null;
}

const REORDERABLE_COLUMN_SET = new Set<CampaignListReorderableColumnId>([
  'name',
  ...CAMPAIGN_LIST_MIDDLE_COLUMNS,
]);

export function isCampaignListMiddleColumnId(
  id: CampaignListColumnId,
): id is CampaignListMiddleColumnId {
  return MIDDLE_COLUMN_SET.has(id as CampaignListMiddleColumnId);
}

export function isCampaignListColumnDraggable(
  id: CampaignListColumnId,
): id is CampaignListReorderableColumnId {
  return REORDERABLE_COLUMN_SET.has(id as CampaignListReorderableColumnId);
}

export function isCampaignListColumnResizable(id: CampaignListColumnId): boolean {
  return id !== 'select';
}

const NUMERIC_MIDDLE_COLUMNS = new Set<CampaignListMiddleColumnId>([
  'clicks',
  'impressions',
  'ctr',
  'unique_clicks',
  'lp_clicks',
  'lp_views',
  'lp_ctr',
  'cr',
  'leads',
  'approved',
  'hold_leads',
  'rejected_leads',
  'approve_rate',
  'epc',
  'cpc',
  'cpa',
  'ecpa',
  'cpm',
  'blocks',
  'block_pct',
  'bots',
  'bot_pct',
  'revenue',
  'cost',
  'profit',
  'roi',
  'budget_pct',
]);

export function isCampaignListNumericColumn(id: CampaignListColumnId): boolean {
  if (id === 'id') {
    return true;
  }
  return isCampaignListMiddleColumnId(id) && NUMERIC_MIDDLE_COLUMNS.has(id);
}

export function resolveCampaignListColumnWidthPx(
  columnId: CampaignListColumnId,
  localWidths: Readonly<Partial<Record<CampaignListColumnId, number>>>,
): number {
  const width = localWidths[columnId];
  if (width != null && Number.isFinite(width) && width > 0) {
    return clampCampaignListColumnWidthPx(columnId, width);
  }
  return CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[columnId];
}

export function defaultCampaignListColumnPrefs(): CampaignListColumnPrefs {
  return {
    dataColumnOrder: ['name', ...CAMPAIGN_LIST_MIDDLE_COLUMNS],
    hidden: [...CAMPAIGN_LIST_DEFAULT_HIDDEN],
    widthPx: {},
  };
}

export function normalizeMiddleOrder(
  order: ReadonlyArray<string>,
): CampaignListMiddleColumnId[] {
  const seen = new Set<CampaignListMiddleColumnId>();
  const result: CampaignListMiddleColumnId[] = [];

  for (const raw of order) {
    const columnId = migrateLegacyColumnId(raw);
    if (!columnId || columnId === 'name') {
      continue;
    }
    if (seen.has(columnId)) {
      continue;
    }
    seen.add(columnId);
    result.push(columnId);
  }

  for (const columnId of CAMPAIGN_LIST_MIDDLE_COLUMNS) {
    if (!seen.has(columnId)) {
      result.push(columnId);
    }
  }

  return result;
}

export function normalizeDataColumnOrder(
  order: ReadonlyArray<string>,
): CampaignListReorderableColumnId[] {
  const seen = new Set<CampaignListReorderableColumnId>();
  const result: CampaignListReorderableColumnId[] = [];

  for (const raw of order) {
    const columnId = migrateLegacyColumnId(raw);
    if (!columnId) {
      continue;
    }
    if (seen.has(columnId)) {
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

export function normalizeHidden(
  hidden: ReadonlyArray<string>,
): CampaignListMiddleColumnId[] {
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

export function normalizeColumnWidthPx(
  widthPx: unknown,
): Partial<Record<CampaignListColumnId, number>> {
  if (!widthPx || typeof widthPx !== 'object') {
    return {};
  }

  const result: Partial<Record<CampaignListColumnId, number>> = {};
  for (const [rawKey, rawValue] of Object.entries(widthPx)) {
    if (rawKey !== 'select' && rawKey !== 'id' && rawKey !== 'name') {
      const migrated = migrateLegacyColumnId(rawKey);
      if (!migrated || migrated === 'name') {
        continue;
      }
      const columnId = migrated;
      const parsed = Number(rawValue);
      if (!Number.isFinite(parsed) || parsed <= 0) {
        continue;
      }
      result[columnId] = clampCampaignListColumnWidthPx(columnId, parsed);
      continue;
    }
    const columnId = rawKey as CampaignListColumnId;
    const parsed = Number(rawValue);
    if (!Number.isFinite(parsed) || parsed <= 0) {
      continue;
    }
    result[columnId] = clampCampaignListColumnWidthPx(columnId, parsed);
  }
  return result;
}

export function visibleCampaignListColumns(
  prefs: CampaignListColumnPrefs,
): CampaignListColumnId[] {
  const hidden = new Set(prefs.hidden);
  const tail = normalizeDataColumnOrder(prefs.dataColumnOrder).filter((columnId) => {
    if (columnId === 'name') {
      return true;
    }
    return !hidden.has(columnId);
  });
  return ['select', 'id', ...tail];
}

export function middleColumnsForSettings(
  prefs: CampaignListColumnPrefs,
): CampaignListMiddleColumnId[] {
  return normalizeDataColumnOrder(prefs.dataColumnOrder).filter(isCampaignListMiddleColumnId);
}

export function campaignListTableMinWidthPx(columns: ReadonlyArray<CampaignListColumnId>): number {
  return columns.reduce((sum, id) => sum + CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[id], 0);
}

export function mergeCampaignListColumnWidths(
  computed: Readonly<Record<CampaignListColumnId, number>>,
  overrides: Readonly<Partial<Record<CampaignListColumnId, number>>>,
  columns: ReadonlyArray<CampaignListColumnId>,
): Record<CampaignListColumnId, number> {
  const merged = { ...computed };
  for (const columnId of columns) {
    const width =
      overrides[columnId] ??
      merged[columnId] ??
      CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[columnId];
    merged[columnId] = clampCampaignListColumnWidthPx(columnId, width);
  }
  return merged;
}

export function parseCampaignListColumnPrefs(raw: string | null): CampaignListColumnPrefs {
  if (!raw?.trim()) {
    return defaultCampaignListColumnPrefs();
  }

  try {
    const parsed = JSON.parse(raw) as {
      dataColumnOrder?: unknown;
      middleOrder?: unknown;
      hidden?: unknown;
      widthPx?: unknown;
    };

    const dataColumnOrder = Array.isArray(parsed.dataColumnOrder)
      ? normalizeDataColumnOrder(parsed.dataColumnOrder.map(String))
      : normalizeDataColumnOrder([
          'name',
          ...normalizeMiddleOrder(
            Array.isArray(parsed.middleOrder) ? parsed.middleOrder.map(String) : [],
          ),
        ]);

    return {
      dataColumnOrder,
      hidden: normalizeHidden(
        Array.isArray(parsed.hidden) ? parsed.hidden.map(String) : [],
      ),
      widthPx: normalizeColumnWidthPx(parsed.widthPx),
    };
  } catch {
    return defaultCampaignListColumnPrefs();
  }
}

export function serializeCampaignListColumnPrefs(prefs: CampaignListColumnPrefs): string {
  const normalized: CampaignListColumnPrefs = {
    dataColumnOrder: normalizeDataColumnOrder(prefs.dataColumnOrder),
    hidden: normalizeHidden(prefs.hidden),
    widthPx: normalizeColumnWidthPx(prefs.widthPx),
  };
  return JSON.stringify(normalized);
}

export function moveMiddleColumn(
  order: CampaignListMiddleColumnId[],
  fromIndex: number,
  toIndex: number,
): CampaignListMiddleColumnId[] {
  if (fromIndex === toIndex || fromIndex < 0 || toIndex < 0 || fromIndex >= order.length) {
    return order;
  }
  const next = [...order];
  const [item] = next.splice(fromIndex, 1);
  if (!item) {
    return order;
  }
  const clampedTo = Math.max(0, Math.min(toIndex, next.length));
  next.splice(clampedTo, 0, item);
  return next;
}

export function moveDataColumn(
  order: CampaignListReorderableColumnId[],
  draggedId: CampaignListReorderableColumnId,
  targetId: CampaignListReorderableColumnId,
): CampaignListReorderableColumnId[] {
  if (draggedId === targetId) {
    return order;
  }
  const fromIndex = order.indexOf(draggedId);
  const toIndex = order.indexOf(targetId);
  if (fromIndex < 0 || toIndex < 0) {
    return order;
  }
  const next = [...order];
  const [item] = next.splice(fromIndex, 1);
  if (!item) {
    return order;
  }
  next.splice(toIndex, 0, item);
  return normalizeDataColumnOrder(next);
}

export function setMiddleColumnVisible(
  hidden: CampaignListMiddleColumnId[],
  id: CampaignListMiddleColumnId,
  visible: boolean,
): CampaignListMiddleColumnId[] {
  const set = new Set(hidden);
  if (visible) {
    set.delete(id);
  } else {
    set.add(id);
  }
  return normalizeHidden([...set]);
}

export function setCampaignListColumnWidth(
  prefs: CampaignListColumnPrefs,
  columnId: CampaignListColumnId,
  widthPx: number,
): CampaignListColumnPrefs {
  if (!isCampaignListColumnResizable(columnId)) {
    return prefs;
  }
  return {
    ...prefs,
    widthPx: {
      ...prefs.widthPx,
      [columnId]: clampCampaignListColumnWidthPx(columnId, widthPx),
    },
  };
}

export function loadCampaignListColumnPrefs(): CampaignListColumnPrefs {
  if (typeof window === 'undefined') {
    return defaultCampaignListColumnPrefs();
  }
  return parseCampaignListColumnPrefs(
    window.localStorage.getItem(CAMPAIGN_LIST_COLUMNS_STORAGE_KEY),
  );
}

export function saveCampaignListColumnPrefs(prefs: CampaignListColumnPrefs): void {
  if (typeof window === 'undefined') {
    return;
  }
  window.localStorage.setItem(
    CAMPAIGN_LIST_COLUMNS_STORAGE_KEY,
    serializeCampaignListColumnPrefs(prefs),
  );
}
