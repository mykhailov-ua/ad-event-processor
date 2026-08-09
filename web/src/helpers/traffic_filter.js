/** @typedef {{ allowReferrers: string[], blockReferrers: string[], blockEmptyReferrer: boolean }} TrafficFilterRules */

/**
 * Empty traffic filter rules.
 *
 * @returns {TrafficFilterRules}
 */
export function emptyTrafficFilterRules() {
  return { allowReferrers: [], blockReferrers: [], blockEmptyReferrer: false };
}

/**
 * Parse stored referrer_filter JSON (no raw Lua).
 *
 * @param {string} raw
 * @returns {TrafficFilterRules}
 */
export function parseTrafficFilter(raw) {
  if (!raw?.trim()) return emptyTrafficFilterRules();
  try {
    const data = JSON.parse(raw);
    if (data?.version !== 1) return emptyTrafficFilterRules();
    return {
      allowReferrers: Array.isArray(data.allow_referrers) ? data.allow_referrers.map(String) : [],
      blockReferrers: Array.isArray(data.block_referrers) ? data.block_referrers.map(String) : [],
      blockEmptyReferrer: Boolean(data.block_empty_referrer),
    };
  } catch {
    return emptyTrafficFilterRules();
  }
}

/**
 * Serialize traffic filter rules for campaigns.referrer_filter.
 *
 * @param {TrafficFilterRules} rules
 * @returns {string}
 */
export function serializeTrafficFilter(rules) {
  return JSON.stringify({
    version: 1,
    allow_referrers: rules.allowReferrers.filter(Boolean),
    block_referrers: rules.blockReferrers.filter(Boolean),
    block_empty_referrer: rules.blockEmptyReferrer,
  });
}
