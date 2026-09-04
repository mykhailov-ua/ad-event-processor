import type { CSSProperties } from 'react';

export function computeTooltipCoords(
  rect: DOMRect,
  side: 'top' | 'bottom',
  align: 'center' | 'start' | 'end',
  sideOffset: number,
): CSSProperties {
  const top = side === 'bottom' ? rect.bottom + sideOffset : rect.top - sideOffset;

  if (align === 'start') {
    return {
      position: 'fixed',
      left: rect.left,
      top,
      transform: side === 'bottom' ? undefined : 'translateY(-100%)',
      zIndex: 50,
    };
  }

  if (align === 'end') {
    return {
      position: 'fixed',
      left: rect.right,
      top,
      transform: side === 'bottom' ? 'translateX(-100%)' : 'translate(-100%, -100%)',
      zIndex: 50,
    };
  }

  return {
    position: 'fixed',
    left: rect.left + rect.width / 2,
    top,
    transform: side === 'bottom' ? 'translateX(-50%)' : 'translate(-50%, -100%)',
    zIndex: 50,
  };
}
