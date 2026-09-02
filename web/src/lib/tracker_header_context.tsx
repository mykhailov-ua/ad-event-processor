import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';

export type TrackerHeaderSearchConfig = {
  value: string;
  onChange: (value: string) => void;
  onApply: () => void;
  disabled?: boolean;
  placeholder?: string;
};

type TrackerHeaderContextValue = {
  search: TrackerHeaderSearchConfig | null;
  setSearch: (search: TrackerHeaderSearchConfig | null) => void;
};

const TrackerHeaderContext = createContext<TrackerHeaderContextValue | null>(null);

export function TrackerHeaderProvider({ children }: { children: ReactNode }) {
  const [search, setSearch] = useState<TrackerHeaderSearchConfig | null>(null);
  const value = useMemo(() => ({ search, setSearch }), [search]);

  return <TrackerHeaderContext.Provider value={value}>{children}</TrackerHeaderContext.Provider>;
}

function useTrackerHeaderContext(): TrackerHeaderContextValue {
  const context = useContext(TrackerHeaderContext);
  if (!context) {
    throw new Error('useTrackerHeaderContext must be used within TrackerHeaderProvider');
  }
  return context;
}

export function useTrackerHeaderSearchRegistration(config: TrackerHeaderSearchConfig) {
  const { setSearch } = useTrackerHeaderContext();

  useEffect(() => {
    setSearch(config);
    return () => setSearch(null);
  }, [
    config,
    setSearch,
    config.disabled,
    config.onApply,
    config.onChange,
    config.placeholder,
    config.value,
  ]);
}

export function useTrackerHeaderSearch(): TrackerHeaderSearchConfig | null {
  return useTrackerHeaderContext().search;
}
