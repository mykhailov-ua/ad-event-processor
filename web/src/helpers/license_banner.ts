export type LicenseInfo = {
  state?: string;
  plan_code?: string;
  valid_until?: string;
  banner_severity?: string;
  renew_days?: number;
  tier_warnings?: string[];
};

export const PILOT_CONVERT_NUDGE_DAYS = 5;

export const LICENSE_SETTINGS_PATH = '/settings/license';

export function isPilotPlan(planCode?: string): boolean {
  return (planCode ?? '').trim().toLowerCase() === 'pilot';
}

export function isPilotConvertNudge(license?: LicenseInfo | null): boolean {
  if (!license?.state) return false;
  const state = license.state.toLowerCase();
  if (state !== 'valid' && state !== 'active') return false;
  if (!isPilotPlan(license.plan_code)) return false;
  if (license.renew_days == null) return false;
  return license.renew_days <= PILOT_CONVERT_NUDGE_DAYS;
}

export function shouldShowLicenseBanner(license?: LicenseInfo | null): boolean {
  if (!license?.state) return false;
  if (isPilotConvertNudge(license)) return true;

  const state = license.state.toLowerCase();
  const warnings = license.tier_warnings ?? [];
  const renewDays = license.renew_days;
  const healthyActive =
    (state === 'valid' || state === 'active') &&
    warnings.length === 0 &&
    (renewDays == null || renewDays > 7);
  return !healthyActive;
}

export type LicenseBannerCTA = {
  href: string;
  label: string;
  external: boolean;
};

export function resolvePilotConvertCTA(supportUrl?: string): LicenseBannerCTA {
  const url = (supportUrl ?? '').trim();
  if (url.startsWith('http://') || url.startsWith('https://') || url.startsWith('tg://')) {
    return { href: url, label: 'Contact vendor', external: true };
  }
  return { href: LICENSE_SETTINGS_PATH, label: 'Upgrade license', external: false };
}

export function buildLicenseBannerParts(license: LicenseInfo): string[] {
  const state = license.state!.toLowerCase();
  const parts: string[] = [];

  if (isPilotConvertNudge(license)) {
    const days = license.renew_days ?? 0;
    parts.push(`Pilot ends in ${days}d — upgrade to Starter to keep tracking live`);
  } else {
    if (state !== 'valid' && state !== 'active') {
      parts.push(`License: ${license.state}`);
    }
    if (license.plan_code) parts.push(`Plan: ${license.plan_code}`);
    if (license.valid_until && (state === 'valid' || state === 'active' || state === 'grace')) {
      parts.push(`until ${license.valid_until}`);
    }
    if (license.renew_days != null && license.renew_days <= 7) {
      parts.push(`renew ${license.renew_days}d`);
    }
  }

  for (const w of license.tier_warnings ?? []) parts.push(w);
  return parts;
}

export function licenseBannerSeverity(license: LicenseInfo): 'error' | 'warning' {
  const state = license.state!.toLowerCase();
  if (license.banner_severity === 'error' || state === 'expired' || state === 'revoked') {
    return 'error';
  }
  return 'warning';
}
