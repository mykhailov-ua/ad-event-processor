import type { FlowPath } from '@/api/types';

export const DEFAULT_FLOW_PATHS_JSON =
  '[{"weight":100,"landers":[],"offers":[]}]';

export function flowPathsToJson(paths: unknown): string {
  if (Array.isArray(paths)) {
    return JSON.stringify(paths, null, 2);
  }
  return DEFAULT_FLOW_PATHS_JSON;
}

export function parseFlowPathsJson(
  raw: string,
): { ok: true; paths: FlowPath[] } | { ok: false; error: string } {
  try {
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      return { ok: false, error: 'Paths must be a JSON array.' };
    }
    return { ok: true, paths: parsed as FlowPath[] };
  } catch {
    return { ok: false, error: 'Paths JSON is invalid.' };
  }
}

function canonicalFlowPaths(paths: FlowPath[]): string {
  const normalized = paths.map((path) => ({
    weight: path.weight ?? 0,
    landers: [...(path.landers ?? [])]
      .map((ref) => ({ lander_id: ref.lander_id, weight: ref.weight ?? 0 }))
      .sort(
        (left, right) =>
          left.lander_id.localeCompare(right.lander_id) || left.weight - right.weight,
      ),
    offers: [...(path.offers ?? [])]
      .map((ref) => ({ offer_id: ref.offer_id, weight: ref.weight ?? 0 }))
      .sort(
        (left, right) =>
          left.offer_id.localeCompare(right.offer_id) || left.weight - right.weight,
      ),
  }));
  return JSON.stringify(normalized);
}

function coerceFlowPaths(value: unknown): FlowPath[] | undefined {
  if (typeof value === 'string') {
    const parsed = parseFlowPathsJson(value);
    return parsed.ok ? parsed.paths : undefined;
  }
  if (Array.isArray(value)) {
    return value as FlowPath[];
  }
  return undefined;
}

export function flowPathsEqual(left: unknown, right: unknown): boolean {
  const leftPaths = coerceFlowPaths(left);
  const rightPaths = coerceFlowPaths(right);
  if (!leftPaths || !rightPaths) {
    return false;
  }
  return canonicalFlowPaths(leftPaths) === canonicalFlowPaths(rightPaths);
}

export function flowDraftFromSnapshot(flow: {
  name?: string;
  paths?: unknown;
}): { name: string; pathsJson: string } {
  return {
    name: flow.name ?? '',
    pathsJson: flowPathsToJson(flow.paths),
  };
}

export type BuildFlowUpdateResult =
  | { ok: true; body: { name: string; paths: FlowPath[] } }
  | { ok: false; error: string };

export function buildFlowUpdateBody(
  draftName: string,
  draftPathsJson: string,
): BuildFlowUpdateResult {
  const name = draftName.trim();
  if (!name) {
    return { ok: false, error: 'Flow name is required.' };
  }
  const parsedPaths = parseFlowPathsJson(draftPathsJson);
  if (!parsedPaths.ok) {
    return { ok: false, error: parsedPaths.error };
  }
  return { ok: true, body: { name, paths: parsedPaths.paths } };
}
