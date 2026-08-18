export type TrafficFilterRules = {
  allowReferrers: string[];
  blockReferrers: string[];
  blockEmptyReferrer: boolean;
};

export function emptyTrafficFilterRules(): TrafficFilterRules {
  return { allowReferrers: [], blockReferrers: [], blockEmptyReferrer: false };
}

export function parseTrafficFilter(raw: string | null | undefined): TrafficFilterRules {
  if (!raw?.trim()) return emptyTrafficFilterRules();
  try {
    const data: unknown = JSON.parse(raw);
    if (!data || typeof data !== 'object') return emptyTrafficFilterRules();
    const obj = data as Record<string, unknown>;
    if (obj.version !== 1) return emptyTrafficFilterRules();
    return {
      allowReferrers: Array.isArray(obj.allow_referrers) ? obj.allow_referrers.map(String) : [],
      blockReferrers: Array.isArray(obj.block_referrers) ? obj.block_referrers.map(String) : [],
      blockEmptyReferrer: Boolean(obj.block_empty_referrer),
    };
  } catch {
    return emptyTrafficFilterRules();
  }
}

export function serializeTrafficFilter(rules: TrafficFilterRules): string {
  return JSON.stringify({
    version: 1,
    allow_referrers: rules.allowReferrers.filter(Boolean),
    block_referrers: rules.blockReferrers.filter(Boolean),
    block_empty_referrer: rules.blockEmptyReferrer,
  });
}
