import { CountryFlagIcon } from '@/domains/campaigns/list/country_flag_icon';
import { cn } from '@/lib/utils';

const COUNTRY_TONE_BY_CODE: Record<string, string> = {
  US: 'violet',
  CA: 'violet',
  DE: 'amber',
  AT: 'amber',
  CH: 'amber',
  JP: 'rose',
  KR: 'rose',
  CN: 'rose',
  GB: 'sky',
  AU: 'sky',
  NZ: 'sky',
  IE: 'sky',
  BR: 'emerald',
  MX: 'emerald',
  AR: 'emerald',
  CO: 'emerald',
  FR: 'indigo',
  ES: 'indigo',
  IT: 'indigo',
  NL: 'indigo',
  PL: 'cyan',
  UA: 'cyan',
  TR: 'orange',
  IN: 'orange',
  ID: 'orange',
};

function normalizeCountryCode(raw: string): string | null {
  const code = raw.trim().toUpperCase();
  if (!/^[A-Z]{2}$/.test(code)) {
    return null;
  }
  return code;
}

function countryBadgeTone(code: string): string {
  const tone = COUNTRY_TONE_BY_CODE[code] ?? 'neutral';
  return `admin-country-badge--${tone}`;
}

export type CampaignCountryBadgesProps = {
  countries?: readonly string[] | null;
  compact?: boolean;
  max?: number;
  className?: string;
};

export function CampaignCountryBadges({
  countries,
  compact = false,
  max = 4,
  className,
}: CampaignCountryBadgesProps) {
  const codes = Array.from(
    new Set(
      (countries ?? [])
        .map((code) => normalizeCountryCode(code))
        .filter((code): code is string => code != null),
    ),
  );
  if (codes.length === 0) {
    return null;
  }

  const visible = codes.slice(0, max);
  const overflow = codes.length - visible.length;
  const title = codes.join(', ');

  return (
    <span className={cn('admin-country-badges', className)} title={title}>
      {visible.map((code) => (
        <span
          key={code}
          className={cn('admin-country-badge', countryBadgeTone(code))}
          title={code}
        >
          <CountryFlagIcon className="admin-country-badge__flag" code={code} title={code} />
          {!compact ? <span className="admin-country-badge__code">{code}</span> : null}
        </span>
      ))}
      {overflow > 0 ? (
        <span className="admin-country-badge admin-country-badge--more">+{overflow}</span>
      ) : null}
    </span>
  );
}
