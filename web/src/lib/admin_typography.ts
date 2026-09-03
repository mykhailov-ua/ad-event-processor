/** Inter + tnum for numeric columns and KPIs. */
export const ADMIN_TABULAR_CLASS = 'tabular-nums';

/** Alias used in dashboard components. */
export const ADMIN_NUMERIC_CLASS = 'tabular-nums';

/** JetBrains Mono for UUIDs, hashes, URLs, JSON, secrets. */
export const ADMIN_MONO_CLASS = 'font-mono tabular-nums';

export type AdminTypographyRole = 'prose' | 'tabular' | 'mono';

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
