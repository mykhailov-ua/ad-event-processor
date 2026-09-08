// shared destructive confirm + mutation error normalization.
export function confirmDestructiveAction(message: string): boolean {
  return window.confirm(message);
}

export function mutationError(err: unknown): Error {
  return err instanceof Error ? err : new Error(String(err));
}
