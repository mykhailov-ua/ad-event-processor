export type Theme = 'light' | 'dark';

export const THEME_STORAGE_KEY = 'aed-admin-theme';
export const THEME_LIGHT_DEFAULT_MIGRATION_KEY = 'aed-admin-theme-light-default-v2';

export function readStoredTheme(): Theme {
  try {
    migrateLightDefaultTheme();
    const stored = window.localStorage.getItem(THEME_STORAGE_KEY);
    return stored === 'dark' ? 'dark' : 'light';
  } catch {
    return 'light';
  }
}

export function migrateLightDefaultTheme(): void {
  try {
    if (window.localStorage.getItem(THEME_LIGHT_DEFAULT_MIGRATION_KEY) === '1') {
      return;
    }
    window.localStorage.setItem(THEME_STORAGE_KEY, 'light');
    window.localStorage.setItem(THEME_LIGHT_DEFAULT_MIGRATION_KEY, '1');
  } catch {
    // ignore storage errors
  }
}

export function readThemeFromDocument(): Theme {
  const root = document.documentElement;
  if (root.classList.contains('light')) {
    return 'light';
  }
  if (root.classList.contains('dark')) {
    return 'dark';
  }
  return readStoredTheme();
}

export function applyTheme(theme: Theme): void {
  const root = document.documentElement;
  const other: Theme = theme === 'dark' ? 'light' : 'dark';
  if (root.classList.contains(theme) && !root.classList.contains(other)) {
    if (root.style.colorScheme !== theme) {
      root.style.colorScheme = theme;
    }
    return;
  }
  root.classList.remove('light', 'dark');
  root.classList.add(theme);
  root.style.colorScheme = theme;
}

export function persistTheme(theme: Theme): void {
  try {
    window.localStorage.setItem(THEME_STORAGE_KEY, theme);
  } catch {
    // ignore storage errors
  }
}

export function themeToggleLabel(theme: Theme): string {
  return theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme';
}
