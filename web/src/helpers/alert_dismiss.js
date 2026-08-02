/**
 * Check whether an alert banner was dismissed for this session.
 *
 * @param {string} key
 * @returns {boolean}
 */
export function isAlertDismissed(key) {
  try {
    return sessionStorage.getItem(`ui.alert.${key}`) === '1';
  } catch {
    return false;
  }
}

/**
 * Persist alert dismissal for the current browser session.
 *
 * @param {string} key
 * @returns {void}
 */
export function dismissAlert(key) {
  try {
    sessionStorage.setItem(`ui.alert.${key}`, '1');
  } catch {}
}
