import type { SettingsEmptyField } from '@/domains/settings/settings_empty';

const PROFILE_LABELS: Record<string, string> = {
  single_vps: 'Single VPS',
  compose_dev: 'Compose dev',
};

const INGRESS_SCHEMA_LABELS: Record<string, string> = {
  ad_event_processor_native: 'Native ingest',
  openrtb_3: 'OpenRTB 3',
};

const LICENSE_PLAN_LABELS: Record<string, string> = {
  pilot: 'Pilot',
  starter: 'Starter',
  growth: 'Growth',
  enterprise: 'Enterprise',
};

const TOKEN_LABELS: Record<string, string> = {
  api: 'API',
  dev: 'dev',
  rtb: 'RTB',
  usd: 'USD',
  utc: 'UTC',
  vps: 'VPS',
  xdp: 'XDP',
};

const DISPLAY_VALUE_FIELDS = new Set<SettingsEmptyField>([
  'profile',
  'ingress_schema',
  'license_plan',
]);

export function usesSettingsDisplayValue(field: SettingsEmptyField): boolean {
  return DISPLAY_VALUE_FIELDS.has(field);
}

function titleCaseToken(token: string): string {
  const lower = token.toLowerCase();
  if (TOKEN_LABELS[lower]) {
    return TOKEN_LABELS[lower];
  }
  if (lower === 'openrtb') {
    return 'OpenRTB';
  }
  return lower.charAt(0).toUpperCase() + lower.slice(1);
}

export function humanizeSettingsSlug(slug: string): string {
  return slug
    .split('_')
    .filter(Boolean)
    .map(titleCaseToken)
    .join(' ');
}

export function formatSettingsDisplayValue(value: string, field: SettingsEmptyField): string {
  const trimmed = value.trim();
  if (!trimmed) {
    return '';
  }

  switch (field) {
    case 'profile':
      return PROFILE_LABELS[trimmed] ?? humanizeSettingsSlug(trimmed);
    case 'ingress_schema':
      return INGRESS_SCHEMA_LABELS[trimmed] ?? humanizeSettingsSlug(trimmed);
    case 'license_plan':
      return LICENSE_PLAN_LABELS[trimmed] ?? humanizeSettingsSlug(trimmed);
    default:
      return trimmed;
  }
}
