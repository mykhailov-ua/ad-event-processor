/** @type {((msg: { title: string, message: string, code?: string }) => void)|null} */
let pushToast = null;

/**
 * @param {(fn: (msg: { title: string, message: string, code?: string }) => void) => void} fn
 */
export function setToastHandler(fn) {
  pushToast = fn;
}

/**
 * @param {{ title: string, message: string, code?: string }} msg
 */
export function pushToastMessage(msg) {
  if (pushToast) pushToast(msg);
}
