export function isAlertDismissed(key: string): boolean {
  try {
    return sessionStorage.getItem(`ui.alert.${key}`) === '1';
  } catch {
    return false;
  }
}

export function dismissAlert(key: string): void {
  try {
    sessionStorage.setItem(`ui.alert.${key}`, '1');
  } catch {}
}
