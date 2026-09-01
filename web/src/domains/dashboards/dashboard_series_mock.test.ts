import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildDashboardMockPortfolio,
  buildDashboardMockSeries,
  DASHBOARD_MOCK_DEFAULT_FROM,
  DASHBOARD_MOCK_DEFAULT_TO,
} from './dashboard_series_mock.ts';

test('buildDashboardMockSeries covers July 1 through September 1', () => {
  const series = buildDashboardMockSeries(DASHBOARD_MOCK_DEFAULT_FROM, DASHBOARD_MOCK_DEFAULT_TO);
  assert.equal(series.length, 63);
  assert.equal(series[0]?.label, '2026-07-01');
  assert.equal(series.at(-1)?.label, '2026-09-01');
  assert.ok((series[0]?.clicks ?? 0) > 0);
  assert.ok((series[0]?.spend_micro ?? 0) > 0);
  assert.ok((series[0]?.revenue_micro ?? 0) > 0);
  assert.ok((series[0]?.profit_micro ?? 0) !== 0);
});

test('buildDashboardMockSeries has realistic economics across the range', () => {
  const series = buildDashboardMockSeries(DASHBOARD_MOCK_DEFAULT_FROM, DASHBOARD_MOCK_DEFAULT_TO);
  const totalCost = series.reduce((sum, point) => sum + (point.spend_micro ?? 0), 0);
  const totalRevenue = series.reduce((sum, point) => sum + (point.revenue_micro ?? 0), 0);
  assert.ok(totalCost > 0);
  assert.ok(totalRevenue > 0);
  assert.ok(totalRevenue > totalCost * 0.7);
});

test('buildDashboardMockPortfolio fills breakdown tables and recent clicks', () => {
  const portfolio = buildDashboardMockPortfolio({
    customer_id: 'cust-demo',
    period: { from: '2026-07-01T00:00:00', to: '2026-09-01T23:59:59' },
  });
  assert.equal(portfolio.series?.length, 63);
  assert.ok((portfolio.breakdowns?.campaigns?.rows?.length ?? 0) >= 5);
  assert.ok((portfolio.breakdowns?.landers?.rows?.length ?? 0) >= 5);
  assert.ok((portfolio.breakdowns?.offers?.rows?.length ?? 0) >= 5);
  assert.ok((portfolio.breakdowns?.sources?.rows?.length ?? 0) >= 5);
  assert.equal(portfolio.recent_clicks?.length, 10);
  assert.ok((portfolio.kpis?.conversions ?? 0) > 0);
  assert.ok((portfolio.kpis?.cost_micro ?? 0) > 0);
  assert.ok((portfolio.kpis?.revenue_micro ?? 0) > 0);
  assert.ok((portfolio.kpis?.cpc_micro ?? 0) > 0);
});
