/**
 * Admin typography roles. Inter is the UI face; numeric alignment uses Inter tnum
 * (tabular lining figures), not a separate monospace face.
 *
 * Policy: web/DESIGN.md#typography
 */

/** Inter + tnum: money, counts, rates, %, dates/times in columns, pagination, KPI tiles. */
export const ADMIN_TABULAR_CLASS = 'admin-tabular-nums';

/** Alias used in dashboard components (same as ADMIN_TABULAR_CLASS). */
export const ADMIN_NUMERIC_CLASS = 'font-numeric';

/**
 * JetBrains Mono: UUIDs, hashes, URLs, JSON/config editors, cron, secrets, code blocks.
 * Do not use for KPI columns - use ADMIN_TABULAR_CLASS instead.
 */
export const ADMIN_MONO_CLASS = 'admin-data-mono';

export type AdminTypographyRole = 'prose' | 'tabular' | 'mono';

/**
 * Data surfaces that get Inter tnum (aligned digits, proportional glyphs elsewhere).
 */
export const ADMIN_TABULAR_DATA_KINDS = [
  'money',
  'micro_amount',
  'count',
  'rate',
  'percent',
  'ratio',
  'display_id',
  'pagination_range',
  'timestamp_column',
  'chart_axis_tick',
  'kpi_value',
  'status_total_count',
] as const;

export type AdminTabularDataKind = (typeof ADMIN_TABULAR_DATA_KINDS)[number];

/**
 * Data surfaces that get JetBrains Mono (fixed-width for scan/compare of opaque strings).
 */
export const ADMIN_MONO_DATA_KINDS = [
  'uuid',
  'hash',
  'url',
  'json',
  'code',
  'cron',
  'secret',
  'file_path',
  'license_token',
  'external_id',
  'env_key',
] as const;

export type AdminMonoDataKind = (typeof ADMIN_MONO_DATA_KINDS)[number];

const ADMIN_LABEL_ABBREVIATIONS = new Map<string, string>([
  ['id', 'ID'],
  ['ctr', 'CTR'],
  ['cr', 'CR'],
  ['cpc', 'CPC'],
  ['cpa', 'CPA'],
  ['ecpa', 'eCPA'],
  ['cpm', 'CPM'],
  ['epc', 'EPC'],
  ['roi', 'ROI'],
  ['lp', 'LP'],
  ['ar', 'AR'],
  ['api', 'API'],
  ['rtb', 'RTB'],
  ['url', 'URL'],
  ['uuid', 'UUID'],
  ['utc', 'UTC'],
  ['asap', 'ASAP'],
  ['ok', 'OK'],
]);

/** Wire enum / slug -> sentence-style label; preserves known abbreviations. */
export function formatAdminEnumLabel(raw: string): string {
  const trimmed = raw.trim();
  if (!trimmed) {
    return '';
  }
  if (/\s/.test(trimmed) && trimmed !== trimmed.toUpperCase()) {
    return trimmed;
  }

  const words = trimmed
    .replace(/-/g, '_')
    .split('_')
    .filter(Boolean)
    .map((token) => {
      const lower = token.toLowerCase();
      const preserved = ADMIN_LABEL_ABBREVIATIONS.get(lower);
      if (preserved) {
        return preserved;
      }
      if (token.length <= 3 && token === token.toUpperCase() && /[A-Z]/.test(token)) {
        return token;
      }
      return lower;
    });

  if (words.length === 0) {
    return '';
  }

  const [first, ...rest] = words;
  const head =
    ADMIN_LABEL_ABBREVIATIONS.get(first.toLowerCase()) ??
    (first.charAt(0).toUpperCase() + first.slice(1).toLowerCase());
  return [head, ...rest].join(' ');
}

export function formatCampaignStatusLabel(status: string, statusLabelOverride?: string): string {
  const source = statusLabelOverride?.trim() || status;
  switch (source.trim().toUpperCase()) {
    case 'ACTIVE':
      return 'Active';
    case 'PAUSED':
      return 'Paused';
    case 'ARCHIVED':
      return 'Archived';
    default:
      return formatAdminEnumLabel(source) || 'Unknown';
  }
}
