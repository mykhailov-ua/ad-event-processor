export const CAMPAIGN_LIST_COLUMNS_STORAGE_KEY = 'aed.campaigns.listColumns.v9';

export const COLUMN_DRAG_MIME = 'application/x-aed-campaign-column';

/** Pinned after select + id; name stays fixed, other data columns follow dataColumnOrder. */
export const CAMPAIGN_LIST_PINNED_DATA_COLUMNS: CampaignListReorderableColumnId[] = ['name'];

/** Default middle-column order after pinned name: status then finance and core KPIs. */
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

/** @deprecated Renamed to hold_leads; migrated on read from localStorage. */
export type LegacyCampaignListMiddleColumnId = CampaignListMiddleColumnId | 'h_leads';

export type CampaignListColumnId = 'select' | 'id' | 'name' | CampaignListMiddleColumnId;

export function isCampaignListPinnedColumn(columnId: CampaignListColumnId): boolean {
  return columnId === 'select' || columnId === 'id' || columnId === 'name';
}

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
  widthPx: Partial<Record<CampaignListColumnId, number>>;
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

export const CAMPAIGN_LIST_SELECTION_COLUMN_WIDTH_PX = 48;

/** Widest status badge label in campaigns directory table probe. */
export const CAMPAIGN_LIST_STATUS_PROBE_LABEL = 'Exhausted';

/** Fixed status column width: probe label + badge padding + cell/tools gutters. */
export const CAMPAIGN_LIST_STATUS_COLUMN_WIDTH_PX = 148;

export const CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX: Record<CampaignListColumnId, number> = {
  select: CAMPAIGN_LIST_SELECTION_COLUMN_WIDTH_PX,
  id: 72,
  name: 200,
  status: CAMPAIGN_LIST_STATUS_COLUMN_WIDTH_PX,
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

const CAMPAIGN_LIST_COLUMN_MAX_WIDTH_PX: Partial<Record<CampaignListColumnId, number>> = {
  select: CAMPAIGN_LIST_SELECTION_COLUMN_WIDTH_PX,
  id: 148,
  name: 320,
  status: CAMPAIGN_LIST_STATUS_COLUMN_WIDTH_PX,
  group: 220,
  owner: 220,
  flow: 160,
  countries: 160,
};

const CAMPAIGN_LIST_COLUMN_DEFAULT_MAX_WIDTH_PX = 160;

const CAMPAIGN_LIST_COLUMN_USER_RESIZE_MAX_WIDTH_PX = 480;

const CAMPAIGN_LIST_COLUMN_USER_RESIZE_MAX_BY_ID: Partial<Record<CampaignListColumnId, number>> = {
  select: CAMPAIGN_LIST_SELECTION_COLUMN_WIDTH_PX,
  name: 640,
  group: 480,
  owner: 480,
};

export function clampCampaignListColumnWidthPx(
  columnId: CampaignListColumnId,
  widthPx: number
): number {
  const minWidth = CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[columnId];
  const maxWidth =
    CAMPAIGN_LIST_COLUMN_MAX_WIDTH_PX[columnId] ?? CAMPAIGN_LIST_COLUMN_DEFAULT_MAX_WIDTH_PX;
  return Math.min(maxWidth, Math.max(minWidth, Math.trunc(widthPx)));
}

export function clampUserResizedCampaignListColumnWidthPx(
  columnId: CampaignListColumnId,
  widthPx: number
): number {
  const minWidth = CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[columnId];
  const maxWidth =
    CAMPAIGN_LIST_COLUMN_USER_RESIZE_MAX_BY_ID[columnId] ??
    CAMPAIGN_LIST_COLUMN_USER_RESIZE_MAX_WIDTH_PX;
  return Math.min(maxWidth, Math.max(minWidth, Math.trunc(widthPx)));
}

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

const REORDERABLE_COLUMN_SET = new Set<CampaignListReorderableColumnId>([
  'name',
  ...CAMPAIGN_LIST_MIDDLE_COLUMNS,
]);

export function isCampaignListMiddleColumnId(
  id: CampaignListColumnId
): id is CampaignListMiddleColumnId {
  return MIDDLE_COLUMN_SET.has(id as CampaignListMiddleColumnId);
}

export function isCampaignListColumnDraggable(
  id: CampaignListColumnId
): id is CampaignListReorderableColumnId {
  return REORDERABLE_COLUMN_SET.has(id as CampaignListReorderableColumnId);
}

export function isCampaignListColumnResizable(
  id: CampaignListColumnId,
  columns?: ReadonlyArray<CampaignListColumnId>
): boolean {
  if (id === 'select' || id === 'id' || id === 'status') {
    return false;
  }
  if (columns != null && columns.length > 0 && columns[columns.length - 1] === id) {
    return false;
  }
  return true;
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
  localWidths: Readonly<Partial<Record<CampaignListColumnId, number>>>
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

export function normalizeMiddleOrder(order: ReadonlyArray<string>): CampaignListMiddleColumnId[] {
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
  order: ReadonlyArray<string>
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

export function normalizeHidden(hidden: ReadonlyArray<string>): CampaignListMiddleColumnId[] {
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
  widthPx: unknown
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
      result[columnId] = clampUserResizedCampaignListColumnWidthPx(columnId, parsed);
      continue;
    }
    const columnId = rawKey as CampaignListColumnId;
    const parsed = Number(rawValue);
    if (!Number.isFinite(parsed) || parsed <= 0) {
      continue;
    }
    result[columnId] = clampUserResizedCampaignListColumnWidthPx(columnId, parsed);
  }
  return result;
}

export function visibleCampaignListColumns(prefs: CampaignListColumnPrefs): CampaignListColumnId[] {
  const hidden = new Set(prefs.hidden);
  const ordered = normalizeDataColumnOrder(prefs.dataColumnOrder);
  const pinned: CampaignListColumnId[] = ['select', 'id'];
  for (const columnId of CAMPAIGN_LIST_PINNED_DATA_COLUMNS) {
    if (columnId === 'name') {
      pinned.push('name');
      continue;
    }
    if (!hidden.has(columnId)) {
      pinned.push(columnId);
    }
  }
  const rest = ordered.filter((columnId) => {
    if (columnId === 'name') {
      return false;
    }
    return !hidden.has(columnId);
  });
  return [...pinned, ...rest];
}

export function visibleMiddleColumnCount(prefs: CampaignListColumnPrefs): number {
  const hidden = new Set(prefs.hidden);
  return CAMPAIGN_LIST_MIDDLE_COLUMNS.filter((columnId) => !hidden.has(columnId)).length;
}

export function middleColumnsForSettings(
  prefs: CampaignListColumnPrefs
): CampaignListMiddleColumnId[] {
  return normalizeDataColumnOrder(prefs.dataColumnOrder).filter(isCampaignListMiddleColumnId);
}

export function campaignListTableMinWidthPx(columns: ReadonlyArray<CampaignListColumnId>): number {
  return columns.reduce((sum, id) => sum + CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[id], 0);
}

const CAMPAIGN_LIST_COLUMN_FALLBACK_WIDTH_PX: Partial<Record<CampaignListColumnId, number>> = {
  countries: 129,
};

export function mergeCampaignListColumnWidths(
  computed: Readonly<Record<CampaignListColumnId, number>>,
  overrides: Readonly<Partial<Record<CampaignListColumnId, number>>>,
  columns: ReadonlyArray<CampaignListColumnId>
): Record<CampaignListColumnId, number> {
  const merged = { ...computed };
  for (const columnId of columns) {
    const resizable = isCampaignListColumnResizable(columnId, columns);
    const overrideWidth = resizable ? overrides[columnId] : undefined;
    const width =
      overrideWidth ??
      merged[columnId] ??
      CAMPAIGN_LIST_COLUMN_FALLBACK_WIDTH_PX[columnId] ??
      CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[columnId];
    merged[columnId] =
      overrideWidth != null
        ? clampUserResizedCampaignListColumnWidthPx(columnId, width)
        : clampCampaignListColumnWidthPx(columnId, width);
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
            Array.isArray(parsed.middleOrder) ? parsed.middleOrder.map(String) : []
          ),
        ]);

    return {
      dataColumnOrder,
      hidden: normalizeHidden(Array.isArray(parsed.hidden) ? parsed.hidden.map(String) : []),
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
  toIndex: number
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
  targetId: CampaignListReorderableColumnId
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
  visible: boolean
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
  columns?: ReadonlyArray<CampaignListColumnId>
): CampaignListColumnPrefs {
  if (!isCampaignListColumnResizable(columnId, columns)) {
    return prefs;
  }
  return {
    ...prefs,
    widthPx: {
      ...prefs.widthPx,
      [columnId]: clampUserResizedCampaignListColumnWidthPx(columnId, widthPx),
    },
  };
}

export function loadCampaignListColumnPrefs(): CampaignListColumnPrefs {
  if (typeof window === 'undefined') {
    return defaultCampaignListColumnPrefs();
  }
  const stored = window.localStorage.getItem(CAMPAIGN_LIST_COLUMNS_STORAGE_KEY);
  if (stored) {
    return parseCampaignListColumnPrefs(stored);
  }
  const legacy = window.localStorage.getItem('aed.campaigns.listColumns.v8');
  if (legacy) {
    const parsed = parseCampaignListColumnPrefs(legacy);
    const migrated: CampaignListColumnPrefs = {
      ...parsed,
      dataColumnOrder: defaultCampaignListColumnPrefs().dataColumnOrder,
    };
    saveCampaignListColumnPrefs(migrated);
    return migrated;
  }
  return defaultCampaignListColumnPrefs();
}

export function saveCampaignListColumnPrefs(prefs: CampaignListColumnPrefs): void {
  if (typeof window === 'undefined') {
    return;
  }
  window.localStorage.setItem(
    CAMPAIGN_LIST_COLUMNS_STORAGE_KEY,
    serializeCampaignListColumnPrefs(prefs)
  );
}
