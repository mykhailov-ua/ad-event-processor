export const CAMPAIGN_LIST_COLUMNS_STORAGE_KEY = 'aed.campaigns.listColumns.v4';

export type CampaignListMiddleColumnId =
  | 'source'
  | 'flows'
  | 'clicks'
  | 'conversions'
  | 'cr'
  | 'revenue'
  | 'cost'
  | 'profit'
  | 'roi'
  | 'group';

export type CampaignListColumnId = 'select' | 'id' | 'name' | CampaignListMiddleColumnId;

export type CampaignListDataColumnId = Exclude<CampaignListColumnId, 'select'>;

export type CampaignListReorderableColumnId = 'name' | CampaignListMiddleColumnId;

export const CAMPAIGN_LIST_MIDDLE_COLUMNS: CampaignListMiddleColumnId[] = [
  'source',
  'cr',
  'flows',
  'clicks',
  'conversions',
  'revenue',
  'cost',
  'profit',
  'roi',
  'group',
];

export const CAMPAIGN_LIST_DEFAULT_HIDDEN: CampaignListMiddleColumnId[] = [];

export type CampaignListColumnPrefs = {
  dataColumnOrder: CampaignListReorderableColumnId[];
  hidden: CampaignListMiddleColumnId[];
  widthPx: Partial<Record<CampaignListColumnId, number>>;
};

export const CAMPAIGN_LIST_COLUMN_LABELS: Record<CampaignListColumnId, string> = {
  select: '',
  id: 'ID',
  name: 'Name',
  source: 'Source',
  flows: 'Flows',
  clicks: 'Clicks',
  conversions: 'Conv.',
  cr: 'CR (sales)',
  revenue: 'Revenue (confirmed)',
  cost: 'Cost',
  profit: 'Profit/Loss (confirmed)',
  roi: 'ROI (confirmed)',
  group: 'Group',
};

export const CAMPAIGN_LIST_SELECTION_COLUMN_WIDTH_PX = 44;

export const CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX: Record<CampaignListColumnId, number> = {
  select: CAMPAIGN_LIST_SELECTION_COLUMN_WIDTH_PX,
  id: 52,
  name: 280,
  source: 120,
  flows: 56,
  clicks: 64,
  conversions: 64,
  cr: 88,
  revenue: 112,
  cost: 88,
  profit: 120,
  roi: 96,
  group: 120,
};

const MIDDLE_COLUMN_SET = new Set<CampaignListMiddleColumnId>(CAMPAIGN_LIST_MIDDLE_COLUMNS);

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
    if (!MIDDLE_COLUMN_SET.has(raw as CampaignListMiddleColumnId)) {
      continue;
    }
    const columnId = raw as CampaignListMiddleColumnId;
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
    if (!REORDERABLE_COLUMN_SET.has(raw as CampaignListReorderableColumnId)) {
      continue;
    }
    const columnId = raw as CampaignListReorderableColumnId;
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
    if (MIDDLE_COLUMN_SET.has(raw as CampaignListMiddleColumnId)) {
      set.add(raw as CampaignListMiddleColumnId);
    }
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
    if (rawKey !== 'select' && rawKey !== 'id' && rawKey !== 'name' && !MIDDLE_COLUMN_SET.has(rawKey as CampaignListMiddleColumnId)) {
      continue;
    }
    const columnId = rawKey as CampaignListColumnId;
    const parsed = Number(rawValue);
    if (!Number.isFinite(parsed) || parsed <= 0) {
      continue;
    }
    result[columnId] = Math.max(
      CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[columnId],
      Math.trunc(parsed),
    );
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
    const override = overrides[columnId];
    if (override != null) {
      merged[columnId] = Math.max(CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[columnId], override);
    }
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
      [columnId]: Math.max(CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[columnId], Math.trunc(widthPx)),
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
