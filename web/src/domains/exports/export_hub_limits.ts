export const EXPORT_HUB_SEARCH_THRESHOLD = 10;

export const EXPORT_HUB_ROW_LIMIT_MIN = 1;

/** Hard server cap (reportjob.ExportRowLimitMax). */
export const EXPORT_HUB_ROW_LIMIT_MAX_DEFAULT = 5_000_000;

/** Default when the operator leaves row limit empty. */
export const EXPORT_HUB_ROW_LIMIT_DEFAULT = 100_000;

export type ExportHubRowLimitBounds = {
  min: number;
  max: number;
  default: number;
};

export type ExportHubRowLimitContext = {
  licenseGated?: boolean;
};

export function resolveExportHubRowLimitBounds(
  context: ExportHubRowLimitContext = {}
): ExportHubRowLimitBounds {
  const max = context.licenseGated ? 1000 : EXPORT_HUB_ROW_LIMIT_MAX_DEFAULT;
  const defaultLimit = Math.min(EXPORT_HUB_ROW_LIMIT_DEFAULT, max);
  return {
    min: EXPORT_HUB_ROW_LIMIT_MIN,
    max,
    default: defaultLimit,
  };
}

export function clampExportHubRowLimit(
  value: number,
  bounds: ExportHubRowLimitBounds
): number {
  if (!Number.isFinite(value)) {
    return bounds.default;
  }
  const rounded = Math.trunc(value);
  return Math.min(bounds.max, Math.max(bounds.min, rounded));
}

export function parseExportHubRowLimitDraft(
  raw: string,
  bounds: ExportHubRowLimitBounds
): number {
  const trimmed = raw.trim();
  if (!trimmed) {
    return bounds.default;
  }
  const parsed = Number.parseInt(trimmed, 10);
  if (!Number.isFinite(parsed)) {
    return bounds.default;
  }
  return clampExportHubRowLimit(parsed, bounds);
}

export type ExportHubRowLimitNormalizeResult = {
  value: number;
  wasInvalid: boolean;
  wasClamped: boolean;
};

export function normalizeExportHubRowLimitDraft(
  raw: string,
  bounds: ExportHubRowLimitBounds
): ExportHubRowLimitNormalizeResult {
  const trimmed = raw.trim();
  if (!trimmed) {
    return { value: bounds.default, wasInvalid: false, wasClamped: false };
  }
  const parsed = Number.parseInt(trimmed, 10);
  if (!Number.isFinite(parsed)) {
    return { value: bounds.default, wasInvalid: true, wasClamped: false };
  }
  const clamped = clampExportHubRowLimit(parsed, bounds);
  return {
    value: clamped,
    wasInvalid: false,
    wasClamped: clamped !== parsed,
  };
}
