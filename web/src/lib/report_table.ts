const ROW_ID_FIELDS = ['id', 'click_id', 'campaign_id', 'customer_id', 'event_id'] as const;

export function deriveColumns(rows: ReadonlyArray<Record<string, unknown>>): string[] {
  const keys = new Set<string>();
  for (const row of rows) {
    for (const key of Object.keys(row)) {
      keys.add(key);
    }
  }
  return Array.from(keys).sort();
}

export function formatMapCell(value: unknown): string {
  if (value == null) {
    return '';
  }
  // Report map rows stringify nested objects on the server (FormatReportCellValue).
  if (typeof value === 'object') {
    return JSON.stringify(value);
  }
  return String(value);
}

export function reportMapRowKey(
  row: Record<string, unknown>,
  columns: readonly string[],
  index: number,
  prefix = '',
): string {
  for (const field of ROW_ID_FIELDS) {
    const value = row[field];
    if (value != null && value !== '') {
      const key = String(value);
      return prefix ? `${prefix}-${key}` : key;
    }
  }

  const firstColumn = columns[0];
  if (firstColumn) {
    const first = row[firstColumn];
    if (first != null && typeof first !== 'object') {
      const key = String(first);
      return prefix ? `${prefix}-${key}` : key;
    }
  }

  const fallback = String(index);
  return prefix ? `${prefix}-${fallback}` : fallback;
}
