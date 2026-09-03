import type { CSSProperties } from 'react';

export function anchorBelowTrigger(
  rect: DOMRect,
  options?: { gap?: number; minWidth?: number },
): CSSProperties {
  const gap = options?.gap ?? 4;
  return {
    position: 'fixed',
    top: rect.bottom + gap,
    left: rect.left,
    minWidth: options?.minWidth ?? rect.width,
    zIndex: 50,
  };
}
