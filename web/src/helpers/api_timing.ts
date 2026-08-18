const durations = new Map<string, number[]>();

const UUID_RE = /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/gi;

export function apiPathTemplate(path: string): string {
  return path.split('?')[0].replace(UUID_RE, '{id}');
}

export function recordApiTiming(path: string, ms: number): void {
  const key = apiPathTemplate(path);
  const bucket = durations.get(key) ?? [];
  bucket.push(ms);
  if (bucket.length > 100) bucket.shift();
  durations.set(key, bucket);
}

function percentile(sorted: number[], p: number): number {
  if (sorted.length === 0) return 0;
  const idx = Math.ceil((p / 100) * sorted.length) - 1;
  return sorted[Math.max(0, idx)];
}

export type ApiTimingPathStats = {
  count: number;
  p50Ms: number;
  p95Ms: number;
};

export type ApiTimingReport = {
  paths: Record<string, ApiTimingPathStats>;
  slowPaths: string[];
};

export function apiTimingReport(): ApiTimingReport {
  const paths: Record<string, ApiTimingPathStats> = {};
  const slowPaths: string[] = [];
  for (const [key, values] of durations.entries()) {
    const sorted = [...values].sort((a, b) => a - b);
    const p50 = Math.round(percentile(sorted, 50));
    const p95 = Math.round(percentile(sorted, 95));
    paths[key] = { count: sorted.length, p50Ms: p50, p95Ms: p95 };
    if (p95 >= 500) slowPaths.push(key);
  }
  return { paths, slowPaths };
}

export function apiTimingReset(): void {
  durations.clear();
}
