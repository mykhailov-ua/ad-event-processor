/**
 * @typedef {Object} IdempotencyScope
 * @property {string} key
 * @property {string} scope
 */

const store = new Map();

/**
 * @returns {string} UUID v4
 */
export function newIdempotencyKey() {
  return crypto.randomUUID();
}

/**
 * @param {string} scope
 * @returns {string} stable key for this scope
 */
export function getOrCreate(scope) {
  if (store.has(scope)) return store.get(scope);
  const key = newIdempotencyKey();
  store.set(scope, key);
  return key;
}

/**
 * @param {string} scope
 */
export function clearScope(scope) {
  store.delete(scope);
}

/**
 */
export function clearAll() {
  store.clear();
}
