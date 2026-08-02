self.onmessage = (e) => {
  const rows = e.data;
  if (!Array.isArray(rows)) {
    self.postMessage([]);
    return;
  }
  const processed = new Array(rows.length);
  for (let i = 0; i < rows.length; i++) {
    const row = rows[i];
    const spend = Number(row.spend_micro ?? 0);
    const revenue = Number(row.revenue_micro ?? 0);
    const roiPct = spend > 0 ? ((revenue - spend) / spend) * 100 : null;
    processed[i] = {
      ...row,
      roi_pct: roiPct,
    };
  }
  self.postMessage(processed);
};
