export function resolvePopoverAlign(
  element: HTMLElement | null,
  viewportWidth = typeof window !== 'undefined' ? window.innerWidth : 0,
  requiredWidth = 280,
  edgePadding = 12,
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
