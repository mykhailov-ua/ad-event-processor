export const CAMPAIGN_LIST_COLUMNS_STORAGE_KEY = 'aed.campaigns.listColumns.v1';

export type CampaignListMiddleColumnId =
  | 'status'
  | 'budget'
  | 'spend'
  | 'clicks'
  | 'conversions'
  | 'pacing'
  | 'customer'
  | 'updated';

export type CampaignListColumnId = 'name' | CampaignListMiddleColumnId | 'actions';

export const CAMPAIGN_LIST_MIDDLE_COLUMNS: CampaignListMiddleColumnId[] = [
  'status',
  'budget',
  'spend',
  'clicks',
  'conversions',
  'pacing',
  'customer',
  'updated',
];

export const CAMPAIGN_LIST_DEFAULT_HIDDEN: CampaignListMiddleColumnId[] = [
  'pacing',
  'customer',
  'updated',
];

export type CampaignListColumnPrefs = {
  middleOrder: CampaignListMiddleColumnId[];
  hidden: CampaignListMiddleColumnId[];
};

export const CAMPAIGN_LIST_COLUMN_LABELS: Record<CampaignListColumnId, string> = {
  name: 'Name',
  status: 'Status',
  budget: 'Budget',
  spend: 'Spend',
  clicks: 'Clicks',
  conversions: 'Conv.',
  pacing: 'Pacing',
  customer: 'Customer',
  updated: 'Updated',
  actions: 'Actions',
};

export const CAMPAIGN_LIST_SELECTION_COLUMN_WIDTH_PX = 44;

export const CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX: Record<CampaignListColumnId, number> = {
  name: 280,
  status: 120,
  budget: 96,
  spend: 128,
  clicks: 56,
  conversions: 56,
  pacing: 64,
  customer: 144,
  updated: 104,
  actions: 72,
};

const MIDDLE_COLUMN_SET = new Set<CampaignListMiddleColumnId>(CAMPAIGN_LIST_MIDDLE_COLUMNS);

export function defaultCampaignListColumnPrefs(): CampaignListColumnPrefs {
  return {
    middleOrder: [...CAMPAIGN_LIST_MIDDLE_COLUMNS],
    hidden: [...CAMPAIGN_LIST_DEFAULT_HIDDEN],
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
    const id = raw as CampaignListMiddleColumnId;
    if (seen.has(id)) {
      continue;
    }
    seen.add(id);
    result.push(id);
  }

  for (const id of CAMPAIGN_LIST_MIDDLE_COLUMNS) {
    if (!seen.has(id)) {
      result.push(id);
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

export function visibleCampaignListColumns(
  prefs: CampaignListColumnPrefs,
): CampaignListColumnId[] {
  const hidden = new Set(prefs.hidden);
  const middle = normalizeMiddleOrder(prefs.middleOrder).filter((id) => !hidden.has(id));
  return ['name', ...middle, 'actions'];
}

export function campaignListTableMinWidthPx(columns: ReadonlyArray<CampaignListColumnId>): number {
  return (
    CAMPAIGN_LIST_SELECTION_COLUMN_WIDTH_PX +
    columns.reduce((sum, id) => sum + CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[id], 0)
  );
}

export function parseCampaignListColumnPrefs(raw: string | null): CampaignListColumnPrefs {
  if (!raw?.trim()) {
    return defaultCampaignListColumnPrefs();
  }

  try {
    const parsed = JSON.parse(raw) as {
      middleOrder?: unknown;
      hidden?: unknown;
    };

    return {
      middleOrder: normalizeMiddleOrder(
        Array.isArray(parsed.middleOrder) ? parsed.middleOrder.map(String) : [],
      ),
      hidden: normalizeHidden(
        Array.isArray(parsed.hidden) ? parsed.hidden.map(String) : [],
      ),
    };
  } catch {
    return defaultCampaignListColumnPrefs();
  }
}

export function serializeCampaignListColumnPrefs(prefs: CampaignListColumnPrefs): string {
  const normalized: CampaignListColumnPrefs = {
    middleOrder: normalizeMiddleOrder(prefs.middleOrder),
    hidden: normalizeHidden(prefs.hidden),
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
