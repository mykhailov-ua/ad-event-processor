const store = new Map<string, string>();

/**
 * Generate a new random idempotency key.
 */
export function newIdempotencyKey(): string {
  return crypto.randomUUID();
}

/**
 * Return a stable idempotency key for the given scope, creating one if needed.
 */
export function getOrCreate(scope: string): string {
  const existing = store.get(scope);
  if (existing) return existing;
  const key = newIdempotencyKey();
  store.set(scope, key);
  return key;
}

/**
 * Drop the in-memory idempotency key for a scope.
 */
export function clearScope(scope: string): void {
  store.delete(scope);
}

/**
 * Clear all in-memory idempotency keys.
 */
export function clearAll(): void {
  store.clear();
}
