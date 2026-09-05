import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';

import {
  applyTheme,
  persistTheme,
  readStoredTheme,
  readThemeFromDocument,
  THEME_STORAGE_KEY,
  type Theme,
} from '@/lib/theme';

type ThemeContextValue = {
  theme: Theme;
  setTheme: (theme: Theme) => void;
};

const ThemeContext = createContext<ThemeContextValue | null>(null);

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<Theme>(() => readThemeFromDocument());

  const setTheme = useCallback((next: Theme) => {
    setThemeState((current) => {
      if (current === next) {
        return current;
      }
      applyTheme(next);
      persistTheme(next);
      return next;
    });
  }, []);

  useEffect(() => {
    const next = readStoredTheme();
    applyTheme(next);
    persistTheme(next);
    setThemeState((current) => (current === next ? current : next));
  }, []);

  useEffect(() => {
    const next = readStoredTheme();
    applyTheme(next);
    persistTheme(next);
    setThemeState((current) => (current === next ? current : next));
  }, []);

  useEffect(() => {
    const onStorage = (event: StorageEvent) => {
      if (event.key !== THEME_STORAGE_KEY) {
        return;
      }
      const next = readStoredTheme();
      setThemeState((current) => {
        if (current === next) {
          return current;
        }
        applyTheme(next);
        persistTheme(next);
        return next;
      });
    };
    window.addEventListener('storage', onStorage);
    return () => window.removeEventListener('storage', onStorage);
  }, []);

  const value = useMemo(() => ({ theme, setTheme }), [theme, setTheme]);

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeContextValue {
  const value = useContext(ThemeContext);
  if (!value) {
    throw new Error('useTheme must be used within ThemeProvider');
  }
  return value;
}
