export function toIsoNow(): string {
  return new Date().toISOString();
}

export function isoDaysAgo(days: number): string {
  const d = new Date();
  d.setDate(d.getDate() - days);
  return d.toISOString();
}

export function isoMonthStart(): string {
  const d = new Date();
  d.setDate(1);
  d.setHours(0, 0, 0, 0);
  return d.toISOString();
}

export type ReportDatePreset = {
  id: string;
  label: string;
  from: () => string;
  to: () => string;
};

export const REPORT_DATE_PRESETS: ReportDatePreset[] = [
  { id: '7d', label: '7 days', from: () => isoDaysAgo(7), to: () => toIsoNow() },
  { id: '30d', label: '30 days', from: () => isoDaysAgo(30), to: () => toIsoNow() },
  { id: 'mtd', label: 'MTD', from: () => isoMonthStart(), to: () => toIsoNow() },
];
