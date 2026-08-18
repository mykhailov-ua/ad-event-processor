export type LicenseInfo = {
  state?: string;
  plan_code?: string;
  valid_until?: string;
  banner_severity?: string;
  renew_days?: number;
  tier_warnings?: string[];
};

export type LicenseBannerProps = {
  license?: LicenseInfo | null;
};

export function LicenseBanner({ license }: LicenseBannerProps) {
  if (!license?.state) return null;

  const state = license.state.toLowerCase();
  const warnings = license.tier_warnings ?? [];
  const renewDays = license.renew_days;
  const healthyActive =
    (state === 'valid' || state === 'active') &&
    warnings.length === 0 &&
    (renewDays == null || renewDays > 7);
  if (healthyActive) return null;

  const severity =
    license.banner_severity ?? (state === 'expired' || state === 'revoked' ? 'error' : 'warning');
  const border = severity === 'error' ? 'var(--error)' : 'var(--warning)';

  const parts: string[] = [];
  if (state !== 'valid' && state !== 'active') {
    parts.push(`License: ${license.state}`);
  }
  if (license.plan_code) parts.push(`Plan: ${license.plan_code}`);
  if (license.valid_until && (state === 'valid' || state === 'active' || state === 'grace')) {
    parts.push(`until ${license.valid_until}`);
  }
  if (renewDays != null && renewDays <= 7) parts.push(`renew ${renewDays}d`);
  for (const w of warnings) parts.push(w);

  return (
    <div
      className="license-banner"
      style={{
        borderColor: border,
        background: `color-mix(in srgb, ${border} 12%, transparent)`,
      }}
    >
      <span>{parts.join(' · ')}</span>
      <a href="/settings/license" style={{ color: 'var(--accent)', fontSize: '12px' }}>
        License
      </a>
    </div>
  );
}
