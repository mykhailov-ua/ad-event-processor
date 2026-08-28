const store = new Map<string, string>();

export function newIdempotencyKey(): string {
  return crypto.randomUUID();
}

export function getOrCreate(scope: string): string {
  const existing = store.get(scope);
  if (existing) return existing;
  const key = newIdempotencyKey();
  store.set(scope, key);
  return key;
}

export function clearScope(scope: string): void {
  store.delete(scope);
}

export function clearAll(): void {
  store.clear();
}
