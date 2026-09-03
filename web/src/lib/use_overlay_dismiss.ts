import { useEffect, type RefObject } from 'react';

export function useOverlayDismiss(
  open: boolean,
  onDismiss: () => void,
  containerRef: RefObject<HTMLElement | null>,
  excludeRefs: Array<RefObject<HTMLElement | null>> = [],
) {
  useEffect(() => {
    if (!open) {
      return;
    }

    const onPointerDown = (event: PointerEvent) => {
      const target = event.target as Node;
      const root = containerRef.current;
      if (root?.contains(target)) {
        return;
      }
      for (const excludeRef of excludeRefs) {
        if (excludeRef.current?.contains(target)) {
          return;
        }
      }
      onDismiss();
    };

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        onDismiss();
      }
    };

    document.addEventListener('pointerdown', onPointerDown, true);
    document.addEventListener('keydown', onKeyDown);
    return () => {
      document.removeEventListener('pointerdown', onPointerDown, true);
      document.removeEventListener('keydown', onKeyDown);
    };
  }, [containerRef, excludeRefs, onDismiss, open]);
}
