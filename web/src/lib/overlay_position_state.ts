import type { CSSProperties } from 'react';

function overlayPositionEqual(a: CSSProperties, b: CSSProperties): boolean {
  const keys = new Set([...Object.keys(a), ...Object.keys(b)]);
  for (const key of keys) {
    if (a[key as keyof CSSProperties] !== b[key as keyof CSSProperties]) {
      return false;
    }
  }
  return true;
}

export function mergeOverlayPosition(prev: CSSProperties, next: CSSProperties): CSSProperties {
  if (overlayPositionEqual(prev, next)) {
    return prev;
  }
  return next;
}
