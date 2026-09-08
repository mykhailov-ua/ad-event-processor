import type { FlowPath } from '@/api/types';

export type FlowPathVisualRow = {
  row_id: string;
  weight: number;
  lander_id: string;
  offer_id: string;
  countries: string;
  devices: string[];
};

const WEIGHT_SUM_TARGET = 100;
const WEIGHT_TOLERANCE = 0.01;

export function newFlowPathRow(): FlowPathVisualRow {
  return {
    row_id: crypto.randomUUID(),
    weight: 100,
    lander_id: '',
    offer_id: '',
    countries: '',
    devices: [],
  };
}

export function flowPathsToVisualRows(paths: FlowPath[]): FlowPathVisualRow[] {
  if (!Array.isArray(paths) || paths.length === 0) {
    return [newFlowPathRow()];
  }
  return paths.map((path) => ({
    row_id: crypto.randomUUID(),
    weight: path.weight ?? 0,
    lander_id: path.landers?.[0]?.lander_id ?? '',
    offer_id: path.offers?.[0]?.offer_id ?? '',
    countries: (path.filters?.countries ?? []).join(', '),
    devices: [...(path.filters?.devices ?? [])],
  }));
}

function parseCountries(raw: string): string[] {
  const out: string[] = [];
  const seen = new Set<string>();
  for (const part of raw.split(/[,\s]+/)) {
    const code = part.trim().toUpperCase();
    if (code.length !== 2 || seen.has(code)) {
      continue;
    }
    seen.add(code);
    out.push(code);
  }
  return out;
}

export function visualRowsToFlowPaths(rows: FlowPathVisualRow[]): FlowPath[] {
  return rows.map((row) => {
    const countries = parseCountries(row.countries);
    const devices = row.devices.filter(Boolean);
    const filters =
      countries.length > 0 || devices.length > 0
        ? { countries, devices }
        : undefined;
    return {
      weight: Math.round(row.weight),
      landers: row.lander_id
        ? [{ lander_id: row.lander_id, weight: 100 }]
        : [],
      offers: row.offer_id ? [{ offer_id: row.offer_id, weight: 100 }] : [],
      filters,
    };
  });
}

export function sumVisualPathWeights(rows: FlowPathVisualRow[]): number {
  return rows.reduce((total, row) => total + (Number.isFinite(row.weight) ? row.weight : 0), 0);
}

export function validateVisualPathWeights(rows: FlowPathVisualRow[]): string | null {
  if (rows.length === 0) {
    return 'At least one path row is required.';
  }
  const sum = sumVisualPathWeights(rows);
  if (Math.abs(sum - WEIGHT_SUM_TARGET) > WEIGHT_TOLERANCE) {
    return `Path weights must sum to 100 (currently ${sum.toFixed(1)}).`;
  }
  for (let index = 0; index < rows.length; index += 1) {
    const row = rows[index];
    if (row.weight <= 0) {
      return `Path ${index + 1} weight must be positive.`;
    }
    if (!row.lander_id) {
      return `Path ${index + 1} requires a lander.`;
    }
    if (!row.offer_id) {
      return `Path ${index + 1} requires an offer.`;
    }
  }
  return null;
}

export function normalizeVisualPathWeights(rows: FlowPathVisualRow[]): FlowPathVisualRow[] {
  const sum = sumVisualPathWeights(rows);
  if (sum <= 0) {
    return rows;
  }
  const scaled = rows.map((row) => ({
    ...row,
    weight: Math.round((row.weight / sum) * WEIGHT_SUM_TARGET),
  }));
  const scaledSum = sumVisualPathWeights(scaled);
  if (scaled.length > 0 && scaledSum !== WEIGHT_SUM_TARGET) {
    const last = scaled.length - 1;
    scaled[last] = {
      ...scaled[last],
      weight: scaled[last].weight + (WEIGHT_SUM_TARGET - scaledSum),
    };
  }
  return scaled;
}

export function applySplitPreset(rows: FlowPathVisualRow[], weights: number[]): FlowPathVisualRow[] {
  if (weights.length === 0) {
    return rows;
  }
  const nextRows = weights.map((weight, index) => {
    const base = rows[index] ?? newFlowPathRow();
    return { ...base, weight };
  });
  return normalizeVisualPathWeights(nextRows);
}

export function buildFlowClickPreviewUrl(campaignId?: string, flowId?: string): string {
  const params = new URLSearchParams({
    type: 'click',
    user_id: 'preview-user',
    click_id: 'preview-click-id',
  });
  if (campaignId?.trim()) {
    params.set('campaign_id', campaignId.trim());
  }
  if (flowId?.trim()) {
    params.set('flow_id', flowId.trim());
  }
  return `/click?${params.toString()}`;
}
