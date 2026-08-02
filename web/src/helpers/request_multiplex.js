const inFlight = new Map();

/**
 * @param {string} key
 * @param {() => Promise<any>} fn
 * @returns {Promise<any>}
 */
export function coalesce(key, fn) {
  if (inFlight.has(key)) return inFlight.get(key);
  const p = fn().finally(() => inFlight.delete(key));
  inFlight.set(key, p);
  return p;
}

/**
 * @param {Array<() => Promise<any>>} tasks
 * @param {number} concurrency
 * @returns {Promise<any[]>}
 */
export async function parallelAll(tasks, concurrency = 6) {
  const results = new Array(tasks.length);
  const active = new Set();
  let idx = 0;

  async function run(i) {
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
  const promises = [];
  for (let i = 0; i < initial; i++) {
    promises.push(run(idx++));
  }
  await Promise.all(promises);
  return results;
}
