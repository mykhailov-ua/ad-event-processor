/**
 * Check whether an alert banner was dismissed for this session.
 */
export function isAlertDismissed(key: string): boolean {
  try {
    return sessionStorage.getItem(`ui.alert.${key}`) === '1';
  } catch {
    return false;
  }
}

/**
 * Persist alert dismissal for the current browser session.
 */
export function dismissAlert(key: string): void {
  try {
    sessionStorage.setItem(`ui.alert.${key}`, '1');
  } catch {
    // ignore quota / private mode
  }
}
