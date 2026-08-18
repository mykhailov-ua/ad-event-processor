const ROW_WORKER_THRESHOLD = 500;
export const MAX_REPORT_ROWS = 2000;
const DISPLAY_ROW_WINDOW = 250;

export type ReportRow = {
  spend_micro?: number;
  revenue_micro?: number;
  roi_pct?: number | null;
  [key: string]: unknown;
};

export type VisibleReportRows = {
  visible: ReportRow[];
  truncated: number;
};

let aggregateWorker: Worker | null = null;

function getAggregateWorker(): Worker | null {
  if (typeof Worker === 'undefined') return null;
  if (!aggregateWorker) {
    aggregateWorker = new Worker('/src/workers/report_aggregate.worker.js', { type: 'module' });
  }
  return aggregateWorker;
}

export function appendReportRows(existing: ReportRow[], batch: ReportRow[]): ReportRow[] {
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

export function enrichReportRowsInPlace(rows: ReportRow[]): ReportRow[] {
  for (let i = 0; i < rows.length; i++) {
    const row = rows[i];
    const spend = Number(row.spend_micro ?? 0);
    const revenue = Number(row.revenue_micro ?? 0);
    row.roi_pct = spend > 0 ? ((revenue - spend) / spend) * 100 : null;
  }
  return rows;
}

function enrichRowsWithWorker(rows: ReportRow[]): Promise<ReportRow[]> {
  const worker = getAggregateWorker();
  if (!worker) return Promise.resolve(enrichReportRowsInPlace(rows));
  return new Promise((resolve) => {
    const onMessage = (e: MessageEvent) => {
      worker.removeEventListener('message', onMessage);
      worker.removeEventListener('error', onError);
      resolve(e.data as ReportRow[]);
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

export function mergeReportRows(existing: ReportRow[], batch: ReportRow[]): Promise<ReportRow[]> {
  const merged = appendReportRows(existing, batch);
  if (merged.length <= ROW_WORKER_THRESHOLD) {
    return Promise.resolve(enrichReportRowsInPlace(merged));
  }
  return enrichRowsWithWorker(merged);
}

export function visibleReportRows(rows: ReportRow[]): VisibleReportRows {
  if (rows.length <= DISPLAY_ROW_WINDOW) {
    return { visible: rows, truncated: 0 };
  }
  return {
    visible: rows.slice(rows.length - DISPLAY_ROW_WINDOW),
    truncated: rows.length - DISPLAY_ROW_WINDOW,
  };
}
