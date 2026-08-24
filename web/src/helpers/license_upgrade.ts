/**
 * Whether the license screen should show pilot request or Starter upgrade paths.
 */
export function showLicenseUpgradePath(planCode?: string, state?: string): boolean {
  const plan = (planCode ?? '').trim().toLowerCase();
  const st = (state ?? '').trim().toUpperCase();
  return st === 'UNCONFIGURED' || plan === '' || plan === 'pilot';
}

/**
 * Format max RPS from JWT limits for operator-facing copy (no unlimited hype).
 */
export function formatLicenseRpsCap(maxRps?: number): string {
  if (maxRps == null || maxRps <= 0) {
    return 'see license JWT';
  }
  return `${maxRps.toLocaleString()} max RPS`;
}

/**
 * Resolve external upgrade contact URL: support link first, else license settings.
 */
export function resolveLicenseUpgradeHref(supportUrl?: string): { href: string; external: boolean } {
  const url = (supportUrl ?? '').trim();
  if (url.startsWith('http://') || url.startsWith('https://') || url.startsWith('tg://')) {
    return { href: url, external: true };
  }
  return { href: '/settings/license', external: false };
}
