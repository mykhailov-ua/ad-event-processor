/**
 * @param {string} key
 */
export function isAlertDismissed(key) {
  try {
    return sessionStorage.getItem(`ui.alert.${key}`) === '1';
  } catch {
    return false;
  }
}

/**
 * @param {string} key
 */
export function dismissAlert(key) {
  try {
    sessionStorage.setItem(`ui.alert.${key}`, '1');
  } catch {}
}
