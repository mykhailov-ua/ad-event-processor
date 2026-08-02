/**
 * @typedef {Object} IdempotencyScope
 * @property {string} key
 * @property {string} scope
 */

const store = new Map();

/**
 * Generate a new random idempotency key.
 *
 * @returns {string} UUID v4
 */
export function newIdempotencyKey() {
  return crypto.randomUUID();
}

/**
 * Return a stable idempotency key for the given scope, creating one if needed.
 *
 * @param {string} scope
 * @returns {string}
 */
export function getOrCreate(scope) {
  if (store.has(scope)) return store.get(scope);
  const key = newIdempotencyKey();
  store.set(scope, key);
  return key;
}

/**
 * Drop the in-memory idempotency key for a scope.
 *
 * @param {string} scope
 * @returns {void}
 */
export function clearScope(scope) {
  store.delete(scope);
}

/**
 * Clear all in-memory idempotency keys.
 *
 * @returns {void}
 */
export function clearAll() {
  store.clear();
}
