import { useState } from 'react';

import {
  COUNTRY_FLAG_HEIGHT,
  COUNTRY_FLAG_WIDTH,
  countryFlagAssetPath,
  normalizeCountryFlagCode,
} from '@/domains/campaigns/list/country_flag_assets';
import { cn } from '@/lib/utils';

export type CountryFlagIconProps = {
  code: string;
  className?: string;
  title?: string;
};

export function CountryFlagIcon({ code, className, title }: CountryFlagIconProps) {
  const [broken, setBroken] = useState(false);
  const normalized = normalizeCountryFlagCode(code) ?? code.trim().toUpperCase();
  const src = countryFlagAssetPath(code);

  if (!src || broken) {
    return (
      <span
        aria-hidden
        className={cn('admin-country-flag admin-country-flag--fallback', className)}
        title={title ?? normalized}
      >
        {normalized.slice(0, 1)}
      </span>
    );
  }

  return (
    <img
      alt=""
      className={cn('admin-country-flag', className)}
      decoding="async"
      height={COUNTRY_FLAG_HEIGHT}
      loading="lazy"
      src={src}
      title={title ?? normalized}
      width={COUNTRY_FLAG_WIDTH}
      onError={() => setBroken(true)}
    />
  );
}
