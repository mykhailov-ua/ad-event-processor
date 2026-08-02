/**
 * @param {Array<object>} existing
 * @param {Array<object>} batch
 * @returns {Promise<Array<object>>}
 */
export function mergeReportRows(existing, batch) {
  const combined = [...existing, ...batch];
  if (combined.length <= 500) return Promise.resolve(combined);
  if (typeof Worker === 'undefined') return Promise.resolve(combined);
  return new Promise((resolve) => {
    const w = new Worker(
      new URL('../workers/report_aggregate.worker.js', import.meta.url),
      { type: 'module' },
    );
    w.onmessage = (e) => { resolve(e.data); w.terminate(); };
    w.onerror = () => { resolve(combined); w.terminate(); };
    w.postMessage(combined);
  });
}
