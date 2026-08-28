#!/usr/bin/env node

import { performance } from 'node:perf_hooks';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
const SRC = join(ROOT, 'src');

const { cn } = await import(pathToFileURL(join(SRC, 'lib/cn.js')).href);
const { formatUsdDecimal } = await import(pathToFileURL(join(SRC, 'helpers/money.js')).href);
const { formatReportCellValue, formatLocaleDateTime } = await import(
  pathToFileURL(join(SRC, 'helpers/format_display.js')).href
);

function makeCampaigns(n) {
  const items = new Array(n);
  for (let i = 0; i < n; i += 1) {
    items[i] = {
      id: `camp-${i}`,
      name: `Campaign ${i}`,
      status: i % 3 === 0 ? 'active' : 'paused',
      budget_limit: String(1_000_000 + i),
      current_spend: String(100_000 + i),
      pacing_mode: 'even',
      updated_at: new Date(Date.now() - i * 60_000).toISOString(),
      customer_id: `cust-${i % 20}`,
    };
  }
  return items;
}

function makeReportRows(n, colCount = 12) {
  const rows = new Array(n);
  for (let i = 0; i < n; i += 1) {
    const row = {};
    for (let c = 0; c < colCount; c += 1) {
      const key = `col_${c}`;
      if (c === colCount - 1 && i % 5 === 0) {
        row[key] = { nested: i, flag: true };
      } else {
        row[key] = i * c;
      }
    }
    rows[i] = row;
  }
  return rows;
}

function makeListPayload(n) {
  return JSON.stringify({ items: makeCampaigns(n), total: n, limit: n, offset: 0 });
}

function statusClass(status) {
  if (!status) return 'statusChip';
  const normalized = status.toLowerCase();
  if (normalized === 'active' || normalized === 'running') {
    return 'statusChip statusActive';
  }
  if (normalized === 'paused') {
    return 'statusChip statusPaused';
  }
  return 'statusChip';
}

function displayStatus(status) {
  if (!status) return '-';
  return status.replace(/_/g, ' ');
}

function formatUpdated(value) {
  return formatLocaleDateTime(value);
}

function formatCellValue(value) {
  return formatReportCellValue(value);
}

function benchHot(name, fn, opts = {}) {
  const iterations = opts.iterations ?? 500;
  const warmup = opts.warmup ?? 50;
  for (let i = 0; i < warmup; i += 1) fn();
  if (global.gc) global.gc();
  const heapBefore = process.memoryUsage().heapUsed;
  const t0 = performance.now();
  for (let i = 0; i < iterations; i += 1) fn();
  const t1 = performance.now();
  const heapAfter = process.memoryUsage().heapUsed;
  const ms = t1 - t0;
  const nsPerOp = (ms * 1e6) / iterations;
  const bytesPerOp = (heapAfter - heapBefore) / iterations;
  return {
    name,
    iterations,
    nsPerOp: Math.round(nsPerOp),
    bytesPerOp: Math.round(bytesPerOp),
    allocsPerOp: bytesPerOp > 0 ? '~' : '0',
  };
}

const ROW_N = 50;
const campaigns = makeCampaigns(ROW_N);
const reportRows = makeReportRows(ROW_N);
const listPayload = makeListPayload(ROW_N);
const selectedIds = new Set(campaigns.map((c) => c.id));

const results = [];

results.push(
  benchHot(
    'cn(3 parts)',
    () => {
      cn('a', false, 'c');
    },
    { iterations: 100_000 }
  )
);

results.push(
  benchHot(
    `grid row format n=${ROW_N} (locale+status)`,
    () => {
      for (let i = 0; i < campaigns.length; i += 1) {
        const c = campaigns[i];
        statusClass(c.status);
        displayStatus(c.status);
        formatUpdated(c.updated_at);
        formatUsdDecimal(c.budget_limit);
      }
    },
    { iterations: 2_000 }
  )
);

results.push(
  benchHot(
    `rowIds map+filter n=${ROW_N}`,
    () => {
      campaigns.map((item) => item.id ?? '').filter(Boolean);
    },
    { iterations: 20_000 }
  )
);

results.push(
  benchHot(
    `Array.from(selectedIds) n=${ROW_N}`,
    () => {
      Array.from(selectedIds);
    },
    { iterations: 50_000 }
  )
);

results.push(
  benchHot(
    `report columns+cells n=${ROW_N} cols=12`,
    () => {
      if (reportRows.length === 0) return;
      const columns = Object.keys(reportRows[0]);
      for (let r = 0; r < reportRows.length; r += 1) {
        const row = reportRows[r];
        for (let c = 0; c < columns.length; c += 1) {
          formatCellValue(row[columns[c]]);
        }
      }
    },
    { iterations: 500 }
  )
);

results.push(
  benchHot(
    `JSON.parse list payload n=${ROW_N}`,
    () => {
      JSON.parse(listPayload);
    },
    { iterations: 500 }
  )
);

results.push(
  benchHot(
    'useResource refetch snapshot (null+loading+data)',
    () => {
      const next = { data: campaigns, loading: false, error: null };
      void next;
    },
    { iterations: 100_000 }
  )
);

console.log('Admin UI hot-path microbench (node --expose-gc)');
console.log('Parity target: react.mdc — minimize ns/op and bytes/op on grid scroll/sort paths');
console.log('');
console.log('| Benchmark | iterations | ns/op | B/op (heap Δ) |');
console.log('|-----------|----------:|------:|--------------:|');
for (const row of results) {
  console.log(`| ${row.name} | ${row.iterations} | ${row.nsPerOp} | ${row.bytesPerOp} |`);
}

const gridRow = results.find((r) => r.name.startsWith('grid row'));
const parseRow = results.find((r) => r.name.startsWith('JSON.parse'));
if (gridRow && parseRow) {
  console.log('');
  console.log(
    `Note: one JSON.parse(n=${ROW_N}) ≈ ${Math.round(parseRow.nsPerOp / (gridRow.nsPerOp / ROW_N))} grid-row-format passes (main-thread parse tax).`
  );
}
