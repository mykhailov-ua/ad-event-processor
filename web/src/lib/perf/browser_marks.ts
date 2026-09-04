/** Optional Performance API marks when ?admin_perf=1 is present (Playwright perf tier). */

export function isAdminPerfEnabled(): boolean {
  if (typeof window === 'undefined') {
    return false;
  }
  const params = new URLSearchParams(window.location.search);
  return params.get('admin_perf') === '1';
}

export function markAdminPerf(name: string): void {
  if (!isAdminPerfEnabled()) {
    return;
  }
  performance.mark(name);
}

export function measureAdminPerf(name: string, startMark: string): number | null {
  if (!isAdminPerfEnabled()) {
    return null;
  }
  const endMark = `${name}-end`;
  performance.mark(endMark);
  performance.measure(name, startMark, endMark);
  const entries = performance.getEntriesByName(name, 'measure');
  const last = entries.at(-1);
  return last?.duration ?? null;
}

declare global {
  interface Window {
    __ADMIN_PERF__?: {
      marks: Record<string, number>;
    };
  }
}

export function publishAdminPerfDuration(name: string, durationMs: number): void {
  if (!isAdminPerfEnabled()) {
    return;
  }
  window.__ADMIN_PERF__ = window.__ADMIN_PERF__ ?? { marks: {} };
  window.__ADMIN_PERF__.marks[name] = durationMs;
}
