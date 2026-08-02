/**
 * @typedef {Object} ConfirmRequest
 * @property {import('./confirm_registry.js').ConfirmEntry} entry
 * @property {string} method
 * @property {string} path
 * @property {string} [title]
 * @property {string} [description]
 */

/** @type {((req: ConfirmRequest) => Promise<boolean>)|null} */
let handler = null;

/**
 * @param {(req: ConfirmRequest) => Promise<boolean>} fn
 */
export function setConfirmHandler(fn) {
  handler = fn;
}

/**
 * @param {ConfirmRequest} req
 * @returns {Promise<boolean>}
 */
export function requestConfirm(req) {
  if (!handler) return Promise.resolve(true);
  return handler(req);
}

export class ConfirmCancelledError extends Error {
  constructor() {
    super('confirm cancelled');
    this.name = 'ConfirmCancelledError';
  }
}
