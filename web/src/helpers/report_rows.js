const ROW_WORKER_THRESHOLD = 500;
export const MAX_REPORT_ROWS = 2000;
const DISPLAY_ROW_WINDOW = 250;

/** @type {Worker|null} */
let aggregateWorker = null;

/**
 * Return a shared report aggregate worker instance.
 *
 * @returns {Worker|null}
 */
function getAggregateWorker() {
  if (typeof Worker === 'undefined') return null;
  if (!aggregateWorker) {
    aggregateWorker = new Worker(
      new URL('../workers/report_aggregate.worker.js', import.meta.url),
      { type: 'module' },
    );
  }
  return aggregateWorker;
}

/**
 * Append report rows in place when the batch is small enough for the main thread.
 *
 * @param {object[]} existing
 * @param {object[]} batch
 * @returns {object[]}
 */
export function appendReportRows(existing, batch) {
  if (batch.length === 0) return existing;
  if (existing.length === 0) return batch.slice();
  const total = existing.length + batch.length;
  if (total > MAX_REPORT_ROWS) {
    const keep = MAX_REPORT_ROWS - existing.length;
    if (keep <= 0) return existing;
    existing.push(...batch.slice(0, keep));
    return existing;
  }
  existing.push(...batch);
  return existing;
}

/**
 * Enrich report rows with roi_pct in place (main-thread, no structured clone).
 *
 * @param {object[]} rows
 * @returns {object[]}
 */
export function enrichReportRowsInPlace(rows) {
  for (let i = 0; i < rows.length; i++) {
    const row = rows[i];
    const spend = Number(row.spend_micro ?? 0);
    const revenue = Number(row.revenue_micro ?? 0);
    row.roi_pct = spend > 0 ? ((revenue - spend) / spend) * 100 : null;
  }
  return rows;
}

/**
 * Run ROI enrichment in a worker when the row count exceeds the threshold.
 *
 * @param {object[]} rows
 * @returns {Promise<object[]>}
 */
function enrichRowsWithWorker(rows) {
  const worker = getAggregateWorker();
  if (!worker) return Promise.resolve(enrichReportRowsInPlace(rows));
  return new Promise((resolve) => {
    const onMessage = (e) => {
      worker.removeEventListener('message', onMessage);
      worker.removeEventListener('error', onError);
      resolve(e.data);
    };
    const onError = () => {
      worker.removeEventListener('message', onMessage);
      worker.removeEventListener('error', onError);
      resolve(enrichReportRowsInPlace(rows));
    };
    worker.addEventListener('message', onMessage);
    worker.addEventListener('error', onError);
    worker.postMessage(rows);
  });
}

/**
 * Merge report row batches and enrich ROI (inline up to ROW_WORKER_THRESHOLD).
 *
 * @param {object[]} existing
 * @param {object[]} batch
 * @returns {Promise<object[]>}
 */
export function mergeReportRows(existing, batch) {
  const merged = appendReportRows(existing, batch);
  if (merged.length <= ROW_WORKER_THRESHOLD) {
    return Promise.resolve(enrichReportRowsInPlace(merged));
  }
  return enrichRowsWithWorker(merged);
}

/**
 * Return the tail slice of rows to render in the DOM table.
 *
 * @param {object[]} rows
 * @returns {{ visible: object[], truncated: number }}
 */
export function visibleReportRows(rows) {
  if (rows.length <= DISPLAY_ROW_WINDOW) {
    return { visible: rows, truncated: 0 };
  }
  return {
    visible: rows.slice(rows.length - DISPLAY_ROW_WINDOW),
    truncated: rows.length - DISPLAY_ROW_WINDOW,
  };
}
