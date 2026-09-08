import { useEffect, useLayoutEffect, useRef, useState } from 'react';

function readContainerWidthPx(node: HTMLElement | null): number {
  return node?.clientWidth ?? 0;
}

export function useDirectoryTableContainerWidth() {
  const hostRef = useRef<HTMLDivElement>(null);
  const [containerWidthPx, setContainerWidthPx] = useState(0);

  useLayoutEffect(() => {
    setContainerWidthPx(readContainerWidthPx(hostRef.current));
  }, []);

  useEffect(() => {
    const node = hostRef.current;
    if (!node) {
      return;
    }

    const observer = new ResizeObserver(() => {
      const nextWidthPx = readContainerWidthPx(node);
      setContainerWidthPx((currentWidthPx) =>
        currentWidthPx === nextWidthPx ? currentWidthPx : nextWidthPx
      );
    });
    observer.observe(node);
    return () => {
      observer.disconnect();
    };
  }, []);

  return { hostRef, containerWidthPx };
}
