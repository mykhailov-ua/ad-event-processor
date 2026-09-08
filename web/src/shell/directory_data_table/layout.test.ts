import assert from 'node:assert/strict';
import test from 'node:test';

import {
  directoryTableNeedsHorizontalScroll,
  directoryTableStretchColumnId,
  directoryTableWidthPx,
  normalizeDirectoryColumnWidths,
  proportionalDirectoryColumnWidths,
  resolveDashboardBreakdownLayoutMaxWidths,
  resolveDirectoryDataTableLayoutWidths,
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

test('directoryTableNeedsHorizontalScroll is false until mins exceed host width', () => {
  const base = { name: 240, clicks: 96 } as const;
  assert.equal(directoryTableNeedsHorizontalScroll(['name', 'clicks'], base, 0), false);
  assert.equal(directoryTableNeedsHorizontalScroll(['name', 'clicks'], base, 500), false);
  assert.equal(directoryTableNeedsHorizontalScroll(['name', 'clicks'], base, 300), true);
});

test('directoryTableStretchColumnId returns the trailing column', () => {
  assert.equal(directoryTableStretchColumnId(['name', 'clicks', 'cost']), 'cost');
});

test('proportionalDirectoryColumnWidths distributes slack across columns', () => {
  const base = { name: 240, clicks: 96, conversions: 96, cost: 112 } as const;
  const stretched = proportionalDirectoryColumnWidths(
    ['name', 'clicks', 'conversions', 'cost'],
    base,
    900,
    resolveDashboardBreakdownLayoutMaxWidths(900)
  );
  const normalized = normalizeDirectoryColumnWidths(
    ['name', 'clicks', 'conversions', 'cost'],
    stretched,
    900
  );
  assert.equal(directoryTableWidthPx(['name', 'clicks', 'conversions', 'cost'], normalized), 900);
  assert.ok(normalized.clicks > base.clicks);
  assert.ok(normalized.cost > base.cost);
  assert.ok(normalized.name < 500);
  assert.ok(normalized.name <= resolveDashboardBreakdownLayoutMaxWidths(900).name!);
});

test('resolveDirectoryDataTableLayoutWidths normalizes stretched widths to container width', () => {
  const base = { name: 240, clicks: 96, conversions: 96, cost: 112 } as const;
  const { layoutWidths, fillContainer } = resolveDirectoryDataTableLayoutWidths(
    ['name', 'clicks', 'conversions', 'cost'],
    base,
    900,
    resolveDashboardBreakdownLayoutMaxWidths(900)
  );
  assert.equal(fillContainer, true);
  assert.equal(directoryTableWidthPx(['name', 'clicks', 'conversions', 'cost'], layoutWidths), 900);
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
