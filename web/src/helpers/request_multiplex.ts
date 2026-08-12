const inFlight = new Map<string, Promise<unknown>>();

export type ParallelSlotError = { error: unknown };

export function isParallelSlotError(v: unknown): v is ParallelSlotError {
  return Boolean(v) && typeof v === 'object' && v !== null && 'error' in v && !('data' in v && 'status' in v);
}

/**
 * Coalesce concurrent requests for the same key into one promise.
 */
export function coalesce<T>(key: string, fn: () => Promise<T>): Promise<T> {
  const existing = inFlight.get(key);
  if (existing) return existing as Promise<T>;
  const p = fn().finally(() => inFlight.delete(key));
  inFlight.set(key, p);
  return p;
}

/**
 * Run tasks with bounded concurrency and collect results in order.
 */
export async function parallelAll<T>(
  tasks: Array<() => Promise<T>>,
  concurrency = 6,
): Promise<Array<T | ParallelSlotError>> {
  const results: Array<T | ParallelSlotError> = new Array(tasks.length);
  const active = new Set<number>();
  let idx = 0;

  /**
   * Run one task slot and chain the next when finished.
   */
  async function run(i: number): Promise<void> {
    active.add(i);
    try {
      results[i] = await tasks[i]();
    } catch (err) {
      results[i] = { error: err };
    } finally {
      active.delete(i);
    }
    if (idx < tasks.length) {
      const next = idx++;
      await run(next);
    }
  }

  const initial = Math.min(concurrency, tasks.length);
  const promises: Promise<void>[] = [];
  for (let i = 0; i < initial; i++) {
    promises.push(run(idx++));
  }
  await Promise.all(promises);
  return results;
}
