import assert from 'node:assert/strict';
import test from 'node:test';

import {
  DEFAULT_CHART_METRIC_IDS,
  DEFAULT_KPI_METRIC_IDS,
  defaultBuyerDashboardPreferences,
  parseBuyerDashboardPreferences,
} from './dashboard_preferences.ts';

test('parseBuyerDashboardPreferences returns defaults for empty input', () => {
  const prefs = parseBuyerDashboardPreferences(null);
  assert.deepEqual(prefs.kpiMetrics, defaultBuyerDashboardPreferences().kpiMetrics);
});

test('defaultBuyerDashboardPreferences keeps seven KPI tiles and five chart lines', () => {
  const prefs = defaultBuyerDashboardPreferences();
  assert.deepEqual(prefs.kpiMetrics, DEFAULT_KPI_METRIC_IDS);
  assert.deepEqual(prefs.chartMetrics, DEFAULT_CHART_METRIC_IDS);
  assert.equal(prefs.kpiMetrics.length, 7);
  assert.equal(prefs.chartMetrics.length, 5);
});

test('parseBuyerDashboardPreferences drops unknown metric ids', () => {
  const raw = JSON.stringify({
    kpiMetrics: ['clicks', 'cpc', 'not_a_metric'],
    chartMetrics: ['revenue', 'bogus'],
    breakdownEntities: ['campaigns', 'invalid'],
    breakdownColumns: ['name', 'cpc', 'roi'],
    recentClickColumns: ['click_id', 'created_at'],
  });
  const prefs = parseBuyerDashboardPreferences(raw);
  assert.deepEqual(prefs.kpiMetrics, ['clicks', 'cpc']);
  assert.deepEqual(prefs.chartMetrics, ['revenue']);
  assert.deepEqual(prefs.breakdownEntities, ['campaigns']);
  assert.deepEqual(prefs.breakdownColumns, ['name', 'cpc', 'roi']);
  assert.deepEqual(prefs.recentClickColumns, ['click_id', 'created_at']);
});

test('parseBuyerDashboardPreferences falls back when every selection is invalid', () => {
  const raw = JSON.stringify({ kpiMetrics: ['bogus'] });
  const prefs = parseBuyerDashboardPreferences(raw);
  assert.deepEqual(prefs.kpiMetrics, defaultBuyerDashboardPreferences().kpiMetrics);
});
