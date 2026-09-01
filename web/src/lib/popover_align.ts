export function resolvePopoverAlign(
  element: HTMLElement | null,
  viewportWidth = typeof window !== 'undefined' ? window.innerWidth : 0,
): 'start' | 'end' {
  if (!element || viewportWidth <= 0) {
    return 'start';
  }
  const rect = element.getBoundingClientRect();
  const spaceRight = viewportWidth - rect.right;
  const spaceLeft = rect.left;
  return spaceRight < spaceLeft ? 'end' : 'start';
}
