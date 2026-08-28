import { parseIsoUnixSeconds } from './iso_time.js';

export type HourlyMetricRow = {
  hour: string;
  impressions?: number;
  clicks?: number;
  conversions?: number;
  [key: string]: string | number | undefined;
};

export type ChartSeries = {
  x: Float64Array;
  y: Float64Array;
  length: number;
};

export function seriesFromHourly(
  hourly: HourlyMetricRow[],
  field: 'impressions' | 'clicks' | 'conversions' | string
): ChartSeries {
  const n = hourly.length;
  const x = new Float64Array(n);
  const y = new Float64Array(n);
  for (let i = 0; i < n; i++) {
    x[i] = parseIsoUnixSeconds(hourly[i].hour);
    const raw = hourly[i][field];
    y[i] = typeof raw === 'number' ? raw : Number(raw ?? 0) || 0;
  }
  return { x, y, length: n };
}
