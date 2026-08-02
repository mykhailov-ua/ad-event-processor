/**
 * Enrich report rows with roi_pct in place to avoid per-row object spreads.
 */
self.onmessage = (e) => {
  const rows = e.data;
  if (!Array.isArray(rows)) {
    self.postMessage([]);
    return;
  }
  for (let i = 0; i < rows.length; i++) {
    const row = rows[i];
    const spend = Number(row.spend_micro ?? 0);
    const revenue = Number(row.revenue_micro ?? 0);
    row.roi_pct = spend > 0 ? ((revenue - spend) / spend) * 100 : null;
  }
  self.postMessage(rows);
};
