import type { CSSProperties } from 'react';

export type FloatingAlign = 'start' | 'center' | 'end';
export type FloatingSide = 'top' | 'bottom';

export function collectScrollContainers(node: HTMLElement | null): Array<HTMLElement | Window> {
  const containers: Array<HTMLElement | Window> = [window];
  let element = node?.parentElement ?? null;
  while (element) {
    const style = window.getComputedStyle(element);
    const overflow = `${style.overflow}${style.overflowY}${style.overflowX}`;
    if (/(auto|scroll|overlay)/.test(overflow)) {
      containers.push(element);
    }
    element = element.parentElement;
  }
  return containers;
}

export function computeFloatingPosition(
  triggerRect: DOMRect,
  contentWidth: number,
  contentHeight: number,
  options: {
    align?: FloatingAlign;
    side?: FloatingSide;
    gap?: number;
    edgePadding?: number;
  } = {},
): CSSProperties {
  const align = options.align ?? 'start';
  const side = options.side ?? 'bottom';
  const gap = options.gap ?? 4;
  const edgePadding = options.edgePadding ?? 12;

  let top =
    side === 'top' ? triggerRect.top - contentHeight - gap : triggerRect.bottom + gap;
  let left = triggerRect.left;
  if (align === 'center') {
    left = triggerRect.left + triggerRect.width / 2 - contentWidth / 2;
  } else if (align === 'end') {
    left = triggerRect.right - contentWidth;
  }
  left = Math.max(edgePadding, Math.min(left, window.innerWidth - contentWidth - edgePadding));
  top = Math.max(edgePadding, Math.min(top, window.innerHeight - contentHeight - edgePadding));

  return {
    position: 'fixed',
    top,
    left,
    zIndex: 60,
  };
}

export function subscribeFloatingPosition(
  trigger: HTMLElement | null,
  onUpdate: () => void,
): () => void {
  if (!trigger) {
    return () => {};
  }

  const containers = collectScrollContainers(trigger);
  const handle = () => {
    onUpdate();
  };

  for (const container of containers) {
    container.addEventListener('scroll', handle, { capture: true, passive: true });
  }
  window.addEventListener('resize', handle, { passive: true });

  return () => {
    for (const container of containers) {
      container.removeEventListener('scroll', handle, { capture: true });
    }
    window.removeEventListener('resize', handle);
  };
}
