/** @type {((msg: { title: string, message: string, code?: string }) => void)|null} */
let pushToast = null;

/**
 * Register the global toast push handler used by pushToastMessage.
 *
 * @param {(fn: (msg: { title: string, message: string, code?: string }) => void) => void} fn
 * @returns {void}
 */
export function setToastHandler(fn) {
  pushToast = fn;
}

/**
 * Enqueue a toast message when a handler is installed.
 *
 * @param {{ title: string, message: string, code?: string }} msg
 * @returns {void}
 */
export function pushToastMessage(msg) {
  if (pushToast) pushToast(msg);
}
