import test from 'node:test';
import assert from 'node:assert/strict';

import {
  campaignListStatsRangeFromDatetimeLocal,
  campaignStatsQueryForRange,
  defaultCampaignListStatsRange,
  isCampaignListStatsRangeWithinLimit,
  legacyStatsRangeFromPreset,
  parseCampaignListStatsRange,
  resolveCampaignListStatsRange,
} from './campaign_list_date_range.ts';
import {
  endOfDayLocalValue,
  fromDatetimeLocalValue,
  startOfDayLocalValue,
} from '@/lib/datetime_range';

test('defaultCampaignListStatsRange returns a bounded 7d window', () => {
  const range = defaultCampaignListStatsRange();
  assert.ok(Date.parse(range.from) < Date.parse(range.to));
});

test('parseCampaignListStatsRange falls back when values are invalid', () => {
  const parsed = parseCampaignListStatsRange(null, null);
  assert.ok(Date.parse(parsed.from) < Date.parse(parsed.to));
  const invalid = parseCampaignListStatsRange('bad', 'worse');
  assert.ok(Date.parse(invalid.from) < Date.parse(invalid.to));
});

test('parseCampaignListStatsRange orders inverted bounds', () => {
  const range = parseCampaignListStatsRange('2026-02-10T00:00:00.000Z', '2026-02-01T00:00:00.000Z');
  assert.equal(range.from, '2026-02-01T00:00:00.000Z');
  assert.equal(range.to, '2026-02-10T00:00:00.000Z');
});

test('campaignStatsQueryForRange maps to API query', () => {
  const range = {
    from: '2026-02-01T00:00:00.000Z',
    to: '2026-02-10T00:00:00.000Z',
  };
  assert.deepEqual(campaignStatsQueryForRange(range), range);
});

test('resolveCampaignListStatsRange prefers explicit from/to over legacy preset', () => {
  const range = resolveCampaignListStatsRange(
    '2026-02-01T00:00:00.000Z',
    '2026-02-10T00:00:00.000Z',
    'today',
  );
  assert.equal(range.from, '2026-02-01T00:00:00.000Z');
  assert.equal(range.to, '2026-02-10T00:00:00.000Z');
});

test('resolveCampaignListStatsRange maps legacy preset when from/to absent', () => {
  const range = resolveCampaignListStatsRange(null, null, 'today');
  assert.ok(Date.parse(range.from) <= Date.parse(range.to));
  assert.equal(legacyStatsRangeFromPreset('all_time'), undefined);
});

test('campaignListStatsRangeFromDatetimeLocal maps picker apply bounds to ISO', () => {
  const fromLocal = startOfDayLocalValue(new Date(2026, 1, 1));
  const toLocal = endOfDayLocalValue(new Date(2026, 1, 10));
  const range = campaignListStatsRangeFromDatetimeLocal(fromLocal, toLocal);
  assert.ok(range);
  assert.equal(fromDatetimeLocalValue(fromLocal), range?.from);
  assert.equal(fromDatetimeLocalValue(toLocal), range?.to);
  assert.deepEqual(campaignStatsQueryForRange(range!), range);
});

test('isCampaignListStatsRangeWithinLimit rejects windows wider than 90 days', () => {
  const ok = isCampaignListStatsRangeWithinLimit({
    from: '2026-01-01T00:00:00.000Z',
    to: '2026-03-01T00:00:00.000Z',
  });
  const wide = isCampaignListStatsRangeWithinLimit({
    from: '2026-01-01T00:00:00.000Z',
    to: '2026-06-01T00:00:00.000Z',
  });
  assert.equal(ok, true);
  assert.equal(wide, false);
});
