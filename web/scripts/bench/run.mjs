#!/usr/bin/env node
/**
 * Admin UI micro-benchmarks (Go benchmem analogue).
 * Run: node --expose-gc web/scripts/bench/run.mjs
 */
import { performance } from 'node:perf_hooks';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
const SRC = join(ROOT, 'src');

const { parseIsoUnixSeconds } = await import(pathToFileURL(join(SRC, 'helpers/iso_time.js')).href);
const { seriesFromHourly } = await import(pathToFileURL(join(SRC, 'helpers/chart_pool.js')).href);
const { appendReportRows } = await import(pathToFileURL(join(SRC, 'helpers/report_rows.js')).href);
const { mapBuyerDashboard, sortPortfolioByDrift, filterPortfolioCampaigns, pacingDriftScore } =
  await import(pathToFileURL(join(SRC, 'models/buyer.js')).href);
const { sortRows, createSortState } = await import(
  pathToFileURL(join(SRC, 'lib/table_sort.js')).href
);

/**
 * Build synthetic hourly metrics for chart benchmarks.
 *
 * @param {number} n
 * @returns {Array<{ hour: string, impressions: number }>}
 */
function makeHourly(n) {
  const rows = new Array(n);
  const base = Date.UTC(2024, 0, 1) / 1000;
  for (let i = 0; i < n; i++) {
    const d = new Date((base + i * 3600) * 1000);
    rows[i] = { hour: d.toISOString(), impressions: (i * 17) % 10000 };
  }
  return rows;
}

/**
 * Build synthetic report rows for merge benchmarks.
 *
 * @param {number} n
 * @returns {object[]}
 */
function makeReportRows(n) {
  const rows = new Array(n);
  for (let i = 0; i < n; i++) {
    rows[i] = {
      placement_id: `p-${i}`,
      campaign_id: `c-${i % 40}`,
      impressions: i,
      clicks: i % 7,
      conversions: i % 3,
      spend_micro: i * 1000,
      revenue_micro: i * 1200,
    };
  }
  return rows;
}

/**
 * Build synthetic campaign rows for sort benchmarks.
 *
 * @param {number} n
 * @returns {object[]}
 */
function makeCampaignRows(n) {
  const rows = new Array(n);
  for (let i = 0; i < n; i++) {
    rows[i] = {
      name: `Campaign ${n - i}`,
      status: i % 2 ? 'ACTIVE' : 'PAUSED',
      budget_limit: i * 10,
      current_spend: i * 3,
      pacing_mode: 'even',
      customer_id: `cust-${i % 5}`,
    };
  }
  return rows;
}

/**
 * Build synthetic buyer dashboard API payload.
 *
 * @param {number} n
 * @returns {object}
 */
function makeBuyerDashboardPayload(n) {
  const campaigns = new Array(n);
  for (let i = 0; i < n; i++) {
    campaigns[i] = {
      id: `c-${i}`,
      name: `Campaign ${i}`,
      status: i % 3 === 0 ? 'PAUSED' : 'ACTIVE',
      pacing_mode: 'even',
      impressions_7d: i * 100,
      clicks_7d: i * 3,
    };
  }
  return {
    customer_id: 'cust-1',
    active: n - Math.floor(n / 3),
    paused: Math.floor(n / 3),
    archived: 0,
    impressions_7d: n * 500,
    clicks_7d: n * 12,
    attention: [],
    campaigns,
  };
}

/**
 * Naive series builder using Date objects per point (pre-optimization baseline).
 *
 * @param {Array<{ hour: string, impressions?: number }>} hourly
 * @returns {{ x: Float64Array, y: Float64Array, length: number }}
 */
function seriesFromHourlyNaive(hourly) {
  const n = hourly.length;
  const x = new Float64Array(n);
  const y = new Float64Array(n);
  for (let i = 0; i < n; i++) {
    x[i] = new Date(hourly[i].hour).getTime() / 1000;
    y[i] = hourly[i].impressions ?? 0;
  }
  return { x, y, length: n };
}

/**
 * Naive report merge using array spread (pre-optimization baseline).
 *
 * @param {object[]} existing
 * @param {object[]} batch
 * @returns {object[]}
 */
function mergeReportRowsNaive(existing, batch) {
  return [...existing, ...batch];
}

/**
 * Naive table sort with separate reverse pass (pre-optimization baseline).
 *
 * @param {object[]} rows
 * @param {string} key
 * @returns {object[]}
 */
function sortRowsNaive(rows, key) {
  const sorted = [...rows].sort((a, b) => String(a[key]).localeCompare(String(b[key])));
  sorted.reverse();
  return sorted;
}

/**
 * Naive buyer dashboard mapping via repeated object spreads (pre-P1 baseline).
 *
 * @param {object} data
 * @returns {object}
 */
function mapBuyerDashboardNaive(data) {
  const campaigns = data?.campaigns ?? [];
  return {
    ...data,
    campaigns: [...campaigns],
  };
}

/**
 * Run a timed benchmark loop and capture heap delta.
 *
 * @param {string} name
 * @param {() => void} fn
 * @param {{ iterations?: number, warmup?: number }} [opts]
 * @returns {{ name: string, iterations: number, ms: number, opsPerSec: number, heapDeltaBytes: number, nsPerOp: number }}
 */
function bench(name, fn, opts = {}) {
  const iterations = opts.iterations ?? 200;
  const warmup = opts.warmup ?? 20;
  for (let i = 0; i < warmup; i++) fn();
  if (global.gc) global.gc();
  const heapBefore = process.memoryUsage().heapUsed;
  const t0 = performance.now();
  for (let i = 0; i < iterations; i++) fn();
  const t1 = performance.now();
  const heapAfter = process.memoryUsage().heapUsed;
  const ms = t1 - t0;
  return {
    name,
    iterations,
    ms: Number(ms.toFixed(3)),
    opsPerSec: Math.round(iterations / (ms / 1000)),
    heapDeltaBytes: heapAfter - heapBefore,
    nsPerOp: Math.round((ms * 1e6) / iterations),
  };
}

const hourly168 = makeHourly(168);
const hourly2048 = makeHourly(2048);
const reportA = makeReportRows(500);
const reportB = makeReportRows(50);
const campaigns50 = makeCampaignRows(50);
const buyerDashboard50 = makeBuyerDashboardPayload(50);
const portfolioRows50 = buyerDashboard50.campaigns;
const sortState = createSortState('name', 'desc');
const accessors = {
  name: (c) => c.name ?? '',
  status: (c) => c.status ?? '',
  budget_limit: (c) => Number(c.budget_limit ?? 0),
};
const sortCache = {};

const benches = [
  bench(
    'parseIsoUnixSeconds x168',
    () => {
      for (let i = 0; i < hourly168.length; i++) parseIsoUnixSeconds(hourly168[i].hour);
    },
    { iterations: 5000 }
  ),
  bench(
    'Date.parse path x168 (baseline)',
    () => {
      for (let i = 0; i < hourly168.length; i++) {
        new Date(hourly168[i].hour).getTime();
      }
    },
    { iterations: 5000 }
  ),
  bench(
    'seriesFromHourly optimized n=168',
    () => {
      seriesFromHourly(hourly168, 'impressions');
    },
    { iterations: 2000 }
  ),
  bench(
    'seriesFromHourly naive n=168',
    () => {
      seriesFromHourlyNaive(hourly168);
    },
    { iterations: 2000 }
  ),
  bench(
    'seriesFromHourly optimized n=2048',
    () => {
      seriesFromHourly(hourly2048, 'impressions');
    },
    { iterations: 400 }
  ),
  bench(
    'seriesFromHourly naive n=2048',
    () => {
      seriesFromHourlyNaive(hourly2048);
    },
    { iterations: 400 }
  ),
  bench(
    'appendReportRows push n=500+50',
    () => {
      appendReportRows(reportA.slice(), reportB);
    },
    { iterations: 300 }
  ),
  bench(
    'merge spread n=500+50 (baseline)',
    () => {
      mergeReportRowsNaive(reportA, reportB);
    },
    { iterations: 300 }
  ),
  bench(
    'mapBuyerDashboard n=50',
    () => {
      mapBuyerDashboard(buyerDashboard50);
    },
    { iterations: 20000 }
  ),
  bench(
    'mapBuyerDashboard naive n=50 (baseline)',
    () => {
      mapBuyerDashboardNaive(buyerDashboard50);
    },
    { iterations: 20000 }
  ),
  bench(
    'filterPortfolioCampaigns n=50',
    () => {
      filterPortfolioCampaigns(portfolioRows50, 'ACTIVE');
    },
    { iterations: 20000 }
  ),
  bench(
    'sortPortfolioByDrift n=50',
    () => {
      sortPortfolioByDrift(portfolioRows50);
    },
    { iterations: 20000 }
  ),
  bench(
    'pacingDriftScore x50',
    () => {
      for (let i = 0; i < portfolioRows50.length; i++) pacingDriftScore(portfolioRows50[i]);
    },
    { iterations: 20000 }
  ),
  bench(
    'sortRows cached n=50 desc',
    () => {
      sortRows(campaigns50, sortState, accessors, sortCache);
    },
    { iterations: 5000 }
  ),
  bench(
    'sortRows naive n=50 desc (baseline)',
    () => {
      sortRowsNaive(campaigns50, 'name');
    },
    { iterations: 5000 }
  ),
];

console.log('Admin UI benchmarks (node --expose-gc)');
console.log('');
console.log('| Benchmark | ns/op | ops/s | heap delta (bytes) |');
console.log('|-----------|------:|------:|---------------:|');
for (const row of benches) {
  console.log(`| ${row.name} | ${row.nsPerOp} | ${row.opsPerSec} | ${row.heapDeltaBytes} |`);
}

const speedup = (after, before) => {
  const a = benches.find((b) => b.name === after);
  const b = benches.find((b) => b.name === before);
  if (!a || !b) return null;
  return (b.nsPerOp / a.nsPerOp).toFixed(2);
};

console.log('');
console.log('Speedups (baseline / optimized):');
console.log(
  `  series n=168: ${speedup('seriesFromHourly optimized n=168', 'seriesFromHourly naive n=168')}x`
);
console.log(
  `  series n=2048: ${speedup('seriesFromHourly optimized n=2048', 'seriesFromHourly naive n=2048')}x`
);
console.log(
  `  report merge: ${speedup('appendReportRows push n=500+50', 'merge spread n=500+50 (baseline)')}x`
);
console.log(
  `  buyer dashboard map: ${speedup('mapBuyerDashboard n=50', 'mapBuyerDashboard naive n=50 (baseline)')}x`
);
console.log(
  `  sort n=50: ${speedup('sortRows cached n=50 desc', 'sortRows naive n=50 desc (baseline)')}x`
);
console.log(
  `  iso parse vs Date: ${speedup('parseIsoUnixSeconds x168', 'Date.parse path x168 (baseline)')}x per field`
);
