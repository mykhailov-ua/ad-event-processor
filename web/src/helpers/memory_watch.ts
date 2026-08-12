const HEAP_WARN_BYTES = 30 * 1024 * 1024;
const HEAP_WINDOW_MS = 30 * 60 * 1000;

type MemoryWatchState = {
  baseline: number;
  last: number;
  max: number;
  startedAt: number;
  warnings: number;
};

type PerformanceWithMemory = Performance & {
  memory?: { usedJSHeapSize?: number };
};

export type MemoryWatchSnapshot = {
  baselineBytes: number;
  lastBytes: number;
  maxBytes: number;
  warnings: number;
};

const state: MemoryWatchState = {
  baseline: 0,
  last: 0,
  max: 0,
  startedAt: Date.now(),
  warnings: 0,
};

/**
 * Read used JS heap size when exposed by the runtime.
 */
function heapBytes(): number {
  if (typeof performance !== 'undefined') {
    const mem = (performance as PerformanceWithMemory).memory?.usedJSHeapSize;
    if (mem) return mem;
  }
  return 0;
}

/**
 * Sample heap on route leave and track monotonic growth.
 */
export function memoryWatchOnRouteLeave(): void {
  const heap = heapBytes();
  if (heap <= 0) return;
  if (state.baseline === 0) state.baseline = heap;
  state.last = heap;
  if (heap > state.max) state.max = heap;
  const elapsed = Date.now() - state.startedAt;
  if (elapsed >= HEAP_WINDOW_MS && heap - state.baseline >= HEAP_WARN_BYTES) {
    state.warnings += 1;
    state.baseline = heap;
    state.startedAt = Date.now();
    if (typeof console !== 'undefined' && console.warn) {
      console.warn(`[admin] heap growth warning: +${Math.round((heap - state.baseline) / 1024 / 1024)} MB`);
    }
  }
}

/**
 * Return heap watch counters for telemetry export.
 */
export function memoryWatchSnapshot(): MemoryWatchSnapshot {
  return {
    baselineBytes: state.baseline,
    lastBytes: state.last,
    maxBytes: state.max,
    warnings: state.warnings,
  };
}

/**
 * Reset heap watch state.
 */
export function memoryWatchReset(): void {
  state.baseline = 0;
  state.last = 0;
  state.max = 0;
  state.startedAt = Date.now();
  state.warnings = 0;
}
