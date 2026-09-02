import { hasFlag } from 'country-flag-icons';

export const COUNTRY_FLAG_WIDTH = 16;
export const COUNTRY_FLAG_HEIGHT = 12;

function normalizeCountryCode(raw: string): string | null {
  const code = raw.trim().toUpperCase();
  if (!/^[A-Z]{2}$/.test(code)) {
    return null;
  }
  return code;
}

export function countryFlagAssetPath(code: string): string | null {
  const normalized = normalizeCountryCode(code);
  if (!normalized || !hasFlag(normalized)) {
    return null;
  }
  return `/src/flags/3x2/${normalized}.svg`;
}

export function normalizeCountryFlagCode(raw: string): string | null {
  return normalizeCountryCode(raw);
}
