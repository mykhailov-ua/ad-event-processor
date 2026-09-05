import {
  createContext,
  useCallback,
  useContext,
  useLayoutEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import { useLocation } from 'react-router-dom';

type BreadcrumbContextValue = {
  segmentLabels: Record<string, string>;
  setSegmentLabel: (segment: string, label: string) => void;
  clearSegmentLabel: (segment: string) => void;
};

const BreadcrumbContext = createContext<BreadcrumbContextValue | null>(null);

export function BreadcrumbProvider({ children }: { children: ReactNode }) {
  const { pathname } = useLocation();
  const [segmentLabels, setSegmentLabels] = useState<Record<string, string>>({});

  useLayoutEffect(() => {
    setSegmentLabels({});
  }, [pathname]);

  const setSegmentLabel = useCallback((segment: string, label: string) => {
    setSegmentLabels((current) => {
      if (current[segment] === label) {
        return current;
      }
      return { ...current, [segment]: label };
    });
  }, []);

  const clearSegmentLabel = useCallback((segment: string) => {
    setSegmentLabels((current) => {
      if (!(segment in current)) {
        return current;
      }
      const next = { ...current };
      delete next[segment];
      return next;
    });
  }, []);

  const value = useMemo(
    () => ({
      segmentLabels,
      setSegmentLabel,
      clearSegmentLabel,
    }),
    [clearSegmentLabel, segmentLabels, setSegmentLabel],
  );

  return <BreadcrumbContext.Provider value={value}>{children}</BreadcrumbContext.Provider>;
}

export function useBreadcrumbSegmentLabel(
  segment: string | undefined,
  label: string | undefined,
) {
  const setSegmentLabel = useContext(BreadcrumbContext)?.setSegmentLabel;
  const clearSegmentLabel = useContext(BreadcrumbContext)?.clearSegmentLabel;

  useLayoutEffect(() => {
    if (!setSegmentLabel || !clearSegmentLabel || !segment) {
      return;
    }
    if (!label) {
      clearSegmentLabel(segment);
      return;
    }
    setSegmentLabel(segment, label);
    return () => {
      clearSegmentLabel(segment);
    };
  }, [clearSegmentLabel, label, segment, setSegmentLabel]);
}

export function useBreadcrumbSegmentLabels() {
  const context = useContext(BreadcrumbContext);
  return context?.segmentLabels ?? {};
}
