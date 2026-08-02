/**
 * Build shareable report query string with tenant customer_id for admins.
 *
 * @param {Record<string, string>} params
 * @returns {string}
 */
export function tenantReportQueryString(params) {
  const qs = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v) qs.set(k, v);
  }
  return qs.toString();
}

/**
 * Merge customer_id into current location for admin share links.
 *
 * @param {string} customerId
 * @returns {void}
 */
export function syncTenantCustomerToUrl(customerId) {
  if (!customerId) return;
  try {
    const url = new URL(window.location.href);
    if (url.searchParams.get('customer_id') === customerId) return;
    url.searchParams.set('customer_id', customerId);
    window.history.replaceState(null, '', url.pathname + url.search);
  } catch {
    // ignore
  }
}
