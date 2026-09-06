// Flip popover/dropdown horizontal align when requiredWidth would clip at the viewport edge.
export function resolvePopoverAlign(
  element: HTMLElement | null,
  viewportWidth = typeof window !== 'undefined' ? window.innerWidth : 0,
  requiredWidth = 280,
  edgePadding = 12
): 'start' | 'end' {
  if (!element || viewportWidth <= 0) {
    return 'start';
  }
  const rect = element.getBoundingClientRect();
  const fitsOpeningRight = rect.left + requiredWidth <= viewportWidth - edgePadding;
  const fitsOpeningLeft = rect.right - requiredWidth >= edgePadding;

  if (!fitsOpeningRight && fitsOpeningLeft) {
    return 'end';
  }
  if (!fitsOpeningLeft && fitsOpeningRight) {
    return 'start';
  }

  const spaceRight = viewportWidth - rect.right;
  const spaceLeft = rect.left;
  return spaceRight < spaceLeft ? 'end' : 'start';
}

/** Flip popover above trigger when requiredHeight would clip below the viewport. */
export function resolvePopoverSide(
  element: HTMLElement | null,
  estimatedHeight = 360,
  edgePadding = 12,
  viewportHeight = typeof window !== 'undefined' ? window.innerHeight : 0
): 'top' | 'bottom' {
  if (!element || viewportHeight <= 0) {
    return 'bottom';
  }
  const rect = element.getBoundingClientRect();
  const spaceBelow = viewportHeight - rect.bottom - edgePadding;
  const spaceAbove = rect.top - edgePadding;
  if (spaceBelow < estimatedHeight && spaceAbove > spaceBelow) {
    return 'top';
  }
  return 'bottom';
}
