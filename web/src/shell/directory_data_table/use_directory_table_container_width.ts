import { useEffect, useRef, useState } from 'react';

export function useDirectoryTableContainerWidth() {
  const hostRef = useRef<HTMLDivElement>(null);
  const [containerWidthPx, setContainerWidthPx] = useState(0);

  useEffect(() => {
    const node = hostRef.current;
    if (!node) {
      return;
    }
    let frame = 0;
    const commit = () => {
      frame = 0;
      setContainerWidthPx(node.clientWidth);
    };
    const schedule = () => {
      if (frame) {
        cancelAnimationFrame(frame);
      }
      frame = requestAnimationFrame(commit);
    };
    schedule();
    const observer = new ResizeObserver(schedule);
    observer.observe(node);
    return () => {
      observer.disconnect();
      if (frame) {
        cancelAnimationFrame(frame);
      }
    };
  }, []);

  return { hostRef, containerWidthPx };
}
