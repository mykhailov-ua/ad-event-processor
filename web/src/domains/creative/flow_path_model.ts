import type { FlowPath } from '@/api/types';
import { newRandomUuid } from '@/lib/uuid';

export type FlowEntityRefRow = {
  ref_id: string;
  entity_id: string;
  weight: number;
};

export type FlowPathRotationMode = 'weighted' | 'unseen' | 'fix_on' | 'sequential';

export const FLOW_PATH_ROTATION_OPTIONS: { value: FlowPathRotationMode; label: string }[] = [
  { value: 'weighted', label: 'Weighted (sticky hash)' },
  { value: 'unseen', label: 'Unseen (rotate until pool exhausted)' },
  { value: 'fix_on', label: 'Fix on (pin first pick per visitor)' },
  { value: 'sequential', label: 'Sequential (top to bottom)' },
];

export type FlowPathVisualRow = {
  row_id: string;
  weight: number;
  rotation_mode: FlowPathRotationMode;
  landers: FlowEntityRefRow[];
  offers: FlowEntityRefRow[];
  countries: string;
  devices: string[];
};

const WEIGHT_SUM_TARGET = 100;
const WEIGHT_TOLERANCE = 0.01;

export function newFlowEntityRef(weight = 100): FlowEntityRefRow {
  return {
    ref_id: newRandomUuid(),
    entity_id: '',
    weight,
  };
}

function normalizeRotationMode(raw: unknown): FlowPathRotationMode {
  if (raw === 'unseen' || raw === 'fix_on' || raw === 'sequential') {
    return raw;
  }
  return 'weighted';
}

export function newFlowPathRow(): FlowPathVisualRow {
  return {
    row_id: newRandomUuid(),
    weight: 100,
    rotation_mode: 'weighted',
    landers: [newFlowEntityRef()],
    offers: [newFlowEntityRef()],
    countries: '',
    devices: [],
  };
}

function entityRefsFromPath(
  refs: Array<{ lander_id?: string; offer_id?: string; weight?: number }> | undefined,
  idField: 'lander_id' | 'offer_id'
): FlowEntityRefRow[] {
  if (!Array.isArray(refs) || refs.length === 0) {
    return [newFlowEntityRef()];
  }
  return refs.map((ref) => ({
    ref_id: newRandomUuid(),
    entity_id: String(ref[idField] ?? ''),
    weight: ref.weight ?? 0,
  }));
}

export function flowPathsToVisualRows(paths: FlowPath[]): FlowPathVisualRow[] {
  if (!Array.isArray(paths) || paths.length === 0) {
    return [newFlowPathRow()];
  }
  return paths.map((path) => ({
    row_id: newRandomUuid(),
    weight: path.weight ?? 0,
    rotation_mode: normalizeRotationMode(path.rotation_mode),
    landers: entityRefsFromPath(path.landers, 'lander_id'),
    offers: entityRefsFromPath(path.offers, 'offer_id'),
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

function entityRefsToWire(
  refs: FlowEntityRefRow[],
  idField: 'lander_id' | 'offer_id'
): Array<{ lander_id: string; weight: number } | { offer_id: string; weight: number }> {
  const out: Array<{ lander_id: string; weight: number } | { offer_id: string; weight: number }> =
    [];
  for (const ref of refs) {
    const entityId = ref.entity_id.trim();
    if (!entityId || ref.weight <= 0) {
      continue;
    }
    if (idField === 'lander_id') {
      out.push({ lander_id: entityId, weight: Math.round(ref.weight) });
    } else {
      out.push({ offer_id: entityId, weight: Math.round(ref.weight) });
    }
  }
  return out;
}

export function visualRowsToFlowPaths(rows: FlowPathVisualRow[]): FlowPath[] {
  return rows.map((row) => {
    const countries = parseCountries(row.countries);
    const devices = row.devices.filter(Boolean);
    const filters = countries.length > 0 || devices.length > 0 ? { countries, devices } : undefined;
    const path: FlowPath = {
      weight: Math.round(row.weight),
      landers: entityRefsToWire(row.landers, 'lander_id') as FlowPath['landers'],
      offers: entityRefsToWire(row.offers, 'offer_id') as FlowPath['offers'],
      filters,
    };
    if (row.rotation_mode !== 'weighted') {
      path.rotation_mode = row.rotation_mode as FlowPath['rotation_mode'];
    }
    return path;
  });
}

export function sumVisualPathWeights(rows: FlowPathVisualRow[]): number {
  return rows.reduce((total, row) => total + (Number.isFinite(row.weight) ? row.weight : 0), 0);
}

export function sumEntityRefWeights(refs: FlowEntityRefRow[]): number {
  return refs.reduce((total, ref) => total + (Number.isFinite(ref.weight) ? ref.weight : 0), 0);
}

function validateEntityRefs(
  refs: FlowEntityRefRow[],
  label: string,
  pathIndex: number
): string | null {
  if (refs.length === 0) {
    return `Path ${pathIndex + 1} requires at least one ${label}.`;
  }
  let hasEntity = false;
  for (let refIndex = 0; refIndex < refs.length; refIndex += 1) {
    const ref = refs[refIndex];
    if (ref.weight <= 0) {
      return `Path ${pathIndex + 1} ${label} ${refIndex + 1} weight must be positive.`;
    }
    if (ref.entity_id.trim()) {
      hasEntity = true;
    }
  }
  if (!hasEntity) {
    return `Path ${pathIndex + 1} requires a ${label}.`;
  }
  const sum = sumEntityRefWeights(refs.filter((ref) => ref.entity_id.trim()));
  if (Math.abs(sum - WEIGHT_SUM_TARGET) > WEIGHT_TOLERANCE) {
    return `Path ${pathIndex + 1} ${label} weights must sum to 100 (currently ${sum.toFixed(1)}).`;
  }
  return null;
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
    const landerError = validateEntityRefs(row.landers, 'lander', index);
    if (landerError) {
      return landerError;
    }
    const offerError = validateEntityRefs(row.offers, 'offer', index);
    if (offerError) {
      return offerError;
    }
  }
  return null;
}

export function normalizeEntityRefWeights(refs: FlowEntityRefRow[]): FlowEntityRefRow[] {
  const active = refs.filter((ref) => ref.entity_id.trim());
  if (active.length === 0) {
    return refs;
  }
  const sum = sumEntityRefWeights(active);
  if (sum <= 0) {
    return refs;
  }
  const scaled = refs.map((ref) => {
    if (!ref.entity_id.trim()) {
      return ref;
    }
    return {
      ...ref,
      weight: Math.round((ref.weight / sum) * WEIGHT_SUM_TARGET),
    };
  });
  const scaledActive = scaled.filter((ref) => ref.entity_id.trim());
  const scaledSum = sumEntityRefWeights(scaledActive);
  if (scaledActive.length > 0 && scaledSum !== WEIGHT_SUM_TARGET) {
    const lastRef = scaledActive[scaledActive.length - 1];
    return scaled.map((ref) =>
      ref.ref_id === lastRef.ref_id
        ? { ...ref, weight: ref.weight + (WEIGHT_SUM_TARGET - scaledSum) }
        : ref
    );
  }
  return scaled;
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

export function applySplitPreset(
  rows: FlowPathVisualRow[],
  weights: number[]
): FlowPathVisualRow[] {
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
