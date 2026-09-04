import type { CampaignStatsQuery } from '@/api/types';
import { fromDatetimeLocalValue } from '@/lib/datetime_range';

export type CampaignListStatsRange = {
  from: string;
  to: string;
};

export const CAMPAIGN_LIST_STATS_MAX_RANGE_MS = 90 * 24 * 60 * 60 * 1000;

export function defaultCampaignListStatsRange(): CampaignListStatsRange {
  const to = new Date();
  const from = new Date(to);
  from.setUTCDate(from.getUTCDate() - 7);
  return { from: from.toISOString(), to: to.toISOString() };
}

export function parseCampaignListStatsRange(
  fromRaw: string | null,
  toRaw: string | null,
): CampaignListStatsRange {
  const fallback = defaultCampaignListStatsRange();
  const from = fromRaw?.trim() || fallback.from;
  const to = toRaw?.trim() || fallback.to;
  const fromMs = Date.parse(from);
  const toMs = Date.parse(to);
  if (!Number.isFinite(fromMs) || !Number.isFinite(toMs)) {
    return fallback;
  }
  const orderedFromMs = Math.min(fromMs, toMs);
  const orderedToMs = Math.max(fromMs, toMs);
  return {
    from: new Date(orderedFromMs).toISOString(),
    to: new Date(orderedToMs).toISOString(),
  };
}

export function campaignStatsQueryForRange(range: CampaignListStatsRange): CampaignStatsQuery {
  return { from: range.from, to: range.to };
}

export function isCampaignListStatsRangeWithinLimit(range: CampaignListStatsRange): boolean {
  const fromMs = Date.parse(range.from);
  const toMs = Date.parse(range.to);
  if (!Number.isFinite(fromMs) || !Number.isFinite(toMs) || toMs <= fromMs) {
    return false;
  }
  return toMs - fromMs <= CAMPAIGN_LIST_STATS_MAX_RANGE_MS;
}

/** Maps DateRangePicker Apply output (datetime-local) to wire ISO bounds. */
export function campaignListStatsRangeFromDatetimeLocal(
  fromLocal: string,
  toLocal: string,
): CampaignListStatsRange | null {
  if (!fromLocal.trim() || !toLocal.trim()) {
    return null;
  }
  const fromIso = fromDatetimeLocalValue(fromLocal);
  const toIso = fromDatetimeLocalValue(toLocal);
  if (!fromIso || !toIso) {
    return null;
  }
  return parseCampaignListStatsRange(fromIso, toIso);
}

/** Legacy preset param; maps to a bounded range when stats_from/stats_to are absent. */
export function legacyStatsRangeFromPreset(
  preset: string | null,
): CampaignListStatsRange | undefined {
  if (!preset || preset === 'all_time') {
    return undefined;
  }

  const now = new Date();
  const todayStart = startOfUtcDay(now);

  if (preset === 'today') {
    return { from: todayStart.toISOString(), to: now.toISOString() };
  }

  if (preset === 'yesterday') {
    const yesterday = new Date(todayStart);
    yesterday.setUTCDate(yesterday.getUTCDate() - 1);
    return {
      from: yesterday.toISOString(),
      to: endOfUtcDay(yesterday).toISOString(),
    };
  }

  if (preset === 'last_7_days') {
    const from = new Date(todayStart);
    from.setUTCDate(from.getUTCDate() - 6);
    return { from: from.toISOString(), to: now.toISOString() };
  }

  if (preset === 'this_month') {
    const monthStart = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1));
    return { from: monthStart.toISOString(), to: now.toISOString() };
  }

  return undefined;
}

export function resolveCampaignListStatsRange(
  fromRaw: string | null,
  toRaw: string | null,
  legacyPresetRaw: string | null,
): CampaignListStatsRange {
  if (fromRaw?.trim() || toRaw?.trim()) {
    return parseCampaignListStatsRange(fromRaw, toRaw);
  }
  return legacyStatsRangeFromPreset(legacyPresetRaw) ?? defaultCampaignListStatsRange();
}

function startOfUtcDay(date: Date): Date {
  return new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate()));
}

function endOfUtcDay(date: Date): Date {
  return new Date(
    Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate(), 23, 59, 59, 999),
  );
}
