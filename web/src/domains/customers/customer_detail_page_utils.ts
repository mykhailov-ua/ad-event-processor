export const CUSTOMER_DETAIL_PAYMENTS_PAGE_LIMIT = 50;
export const CUSTOMER_DETAIL_LEDGER_PAGE_LIMIT = 50;

export function currentCustomerDetailMonthValue(): string {
  const now = new Date();
  const year = now.getUTCFullYear();
  const month = String(now.getUTCMonth() + 1).padStart(2, '0');
  return `${year}-${month}`;
}

// Rejected promise with AbortError name: useResource treats it as skip, not ErrorBlock (RP-3 tab gating).
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
