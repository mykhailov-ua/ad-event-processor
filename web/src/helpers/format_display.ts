export function formatLocaleDateTime(value: string | undefined): string {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '-';
  return date.toLocaleString();
}

export function formatLocaleDate(value: string | undefined): string {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '-';
  return date.toLocaleDateString();
}

export function formatReportCellValue(value: unknown): string {
  if (value === null || value === undefined) return '';
  if (typeof value === 'object') return JSON.stringify(value);
  return String(value);
}

export function reportRowKey(row: Record<string, unknown>, rowIndex: number): string {
  for (const field of ['id', 'campaign_id', 'customer_id', 'click_id', 'ip_hash']) {
    const raw = row[field];
    if (typeof raw === 'string' && raw.length > 0) return raw;
    if (typeof raw === 'number' && Number.isFinite(raw)) return String(raw);
  }
  return `row-${rowIndex}`;
}
