import { ApiError } from '@/api/api_error';
import { toError } from '@/lib/admin_error';
import { validationError } from '@/lib/admin_validation_error';

export const CUSTOMER_DETAIL_PAYMENTS_PAGE_LIMIT = 50;
export const CUSTOMER_DETAIL_LEDGER_PAGE_LIMIT = 50;

const TAX_PROFILE_FIELD_PATTERNS: { pattern: RegExp; field: string }[] = [
  { pattern: /country/i, field: 'country_code' },
  { pattern: /tax[_\s-]?region/i, field: 'tax_region' },
  { pattern: /tax[_\s-]?scheme/i, field: 'tax_scheme' },
  { pattern: /tax[_\s-]?rate|bps/i, field: 'tax_rate_bps' },
];

export function mapTaxProfileSaveError(err: unknown): Error {
  if (err instanceof ApiError && err.status === 400) {
    const message = err.message.trim();
    if (message !== '') {
      for (const { pattern, field } of TAX_PROFILE_FIELD_PATTERNS) {
        if (pattern.test(message)) {
          return validationError(message, { field });
        }
      }
      return validationError(message);
    }
  }
  return toError(err);
}

export function currentCustomerDetailMonthValue(): string {
  const now = new Date();
  const year = now.getUTCFullYear();
  const month = String(now.getUTCMonth() + 1).padStart(2, '0');
  return `${year}-${month}`;
}

// Rejected promise with AbortError name: useResource treats it as skip, not ErrorBlock (tab gating).
export function skipCustomerDetailTabFetch(): Promise<never> {
  return Promise.reject(new DOMException('Skipped', 'AbortError'));
}

export function taxProfileToDraft(profile: {
  country_code?: string;
  tax_region?: string;
  tax_scheme?: string;
  tax_rate_bps?: number;
}) {
  return {
    countryCode: profile.country_code ?? '',
    taxRegion: profile.tax_region ?? '',
    taxScheme: profile.tax_scheme ?? '',
    taxRateBps:
      profile.tax_rate_bps != null && !Number.isNaN(profile.tax_rate_bps)
        ? String(profile.tax_rate_bps)
        : '',
  };
}
