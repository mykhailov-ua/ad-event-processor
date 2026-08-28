import {
  buildLicenseBannerParts,
  isPilotConvertNudge,
  licenseBannerSeverity,
  PILOT_CONVERT_NUDGE_DAYS,
  resolvePilotConvertCTA,
  shouldShowLicenseBanner,
  type LicenseInfo,
} from '../helpers/license_banner.js';

export type { LicenseInfo };

export type LicenseBannerProps = {
  license?: LicenseInfo | null;
  supportUrl?: string;
};

export function LicenseBanner({ license, supportUrl }: LicenseBannerProps) {
  if (!shouldShowLicenseBanner(license)) return null;

  const lic = license!;
  const severity = licenseBannerSeverity(lic);
  const border = severity === 'error' ? 'var(--error)' : 'var(--warning)';
  const parts = buildLicenseBannerParts(lic);
  const pilotNudge = isPilotConvertNudge(lic);
  const cta = pilotNudge ? resolvePilotConvertCTA(supportUrl) : null;

  return (
    <div
      className="license-banner"
      data-testid="license-banner"
      data-pilot-nudge={pilotNudge ? 'true' : 'false'}
      style={{
        borderColor: border,
        background: `color-mix(in srgb, ${border} 12%, transparent)`,
      }}
    >
      <span>{parts.join(' , ')}</span>
      <span style={{ display: 'inline-flex', gap: '12px', marginLeft: '8px' }}>
        {cta ? (
          <a
            href={cta.href}
            target={cta.external ? '_blank' : undefined}
            rel={cta.external ? 'noopener noreferrer' : undefined}
            data-testid="license-banner-pilot-cta"
            style={{ color: 'var(--accent)', fontSize: '12px' }}
          >
            {cta.label}
          </a>
        ) : null}
        <a href="/settings/license" style={{ color: 'var(--accent)', fontSize: '12px' }}>
          License
        </a>
      </span>
    </div>
  );
}

export { PILOT_CONVERT_NUDGE_DAYS };
