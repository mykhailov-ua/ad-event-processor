export function encodeCursor(offset: number): string {
  const raw = String(offset);
  const bytes = new TextEncoder().encode(raw);
  let binary = '';
  for (let i = 0; i < bytes.length; i += 1) {
    binary += String.fromCharCode(bytes[i]!);
  }
  return btoa(binary);
}

export function decodeCursor(cursor: string): number {
  if (!cursor) return 0;
  const binary = atob(cursor);
  const bytes = Uint8Array.from(binary, (c) => c.charCodeAt(0));
  const raw = new TextDecoder().decode(bytes);
  const offset = Number.parseInt(raw, 10);
  if (!Number.isFinite(offset) || offset < 0) {
    throw new Error('invalid cursor');
  }
  return offset;
}
