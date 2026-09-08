import assert from 'node:assert/strict';
import test from 'node:test';

import {
  directoryTableWidthPx,
  proportionalDirectoryColumnWidths,
  resolveDashboardBreakdownLayoutMaxWidths,
  stretchDirectoryColumnWidths,
} from './layout.ts';

test('stretchDirectoryColumnWidths expands stretch column to fill container', () => {
  const base = { name: 200, clicks: 100, cost: 120 } as const;
  const stretched = stretchDirectoryColumnWidths(['name', 'clicks', 'cost'], base, 500, 'name');
  assert.equal(directoryTableWidthPx(['name', 'clicks', 'cost'], stretched), 500);
  assert.equal(stretched.name, 280);
  assert.equal(stretched.clicks, 100);
});

test('stretchDirectoryColumnWidths leaves widths when container is narrower', () => {
  const base = { name: 200, clicks: 100 } as const;
  const stretched = stretchDirectoryColumnWidths(['name', 'clicks'], base, 250, 'name');
  assert.equal(stretched.name, 200);
});

test('proportionalDirectoryColumnWidths distributes slack across columns', () => {
  const base = { name: 240, clicks: 96, conversions: 96, cost: 112 } as const;
  const stretched = proportionalDirectoryColumnWidths(
    ['name', 'clicks', 'conversions', 'cost'],
    base,
    900,
    resolveDashboardBreakdownLayoutMaxWidths(900)
  );
  assert.equal(directoryTableWidthPx(['name', 'clicks', 'conversions', 'cost'], stretched), 900);
  assert.ok(stretched.clicks > base.clicks);
  assert.ok(stretched.cost > base.cost);
  assert.ok(stretched.name < 500);
  assert.ok(stretched.name <= resolveDashboardBreakdownLayoutMaxWidths(900).name!);
});

test('proportionalDirectoryColumnWidths keeps base widths when container is tight', () => {
  const base = { name: 240, clicks: 96 } as const;
  const stretched = proportionalDirectoryColumnWidths(['name', 'clicks'], base, 300);
  assert.equal(stretched.name, 240);
  assert.equal(stretched.clicks, 96);
});

test('proportionalDirectoryColumnWidths uses horizontal scroll when card is narrower than column mins', () => {
  const columns = [
    'name',
    'clicks',
    'unique_clicks',
    'conversions',
    'cost',
    'revenue',
    'profit',
    'roi',
  ] as const;
  const base = {
    name: 220,
    clicks: 72,
    unique_clicks: 228,
    conversions: 120,
    cost: 96,
    revenue: 104,
    profit: 96,
    roi: 80,
  };
  const containerWidthPx = 520;
  const maxWidths = resolveDashboardBreakdownLayoutMaxWidths(containerWidthPx);
  const stretched = proportionalDirectoryColumnWidths(columns, base, containerWidthPx, maxWidths);
  assert.deepEqual(stretched, base);
  assert.ok(directoryTableWidthPx(columns, stretched) > containerWidthPx);
});
