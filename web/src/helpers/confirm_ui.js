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

/** @type {Promise<void>} */
let confirmQueue = Promise.resolve();

/**
 * Register the global confirm-dialog handler.
 *
 * @param {(req: ConfirmRequest) => Promise<boolean>} fn
 * @returns {void}
 */
export function setConfirmHandler(fn) {
  handler = fn;
}

/**
 * Run the confirm flow for a mutation request.
 *
 * @param {ConfirmRequest} req
 * @returns {Promise<boolean>}
 */
export function requestConfirm(req) {
  if (!handler) return Promise.resolve(true);
  const run = confirmQueue.then(() => handler(req));
  confirmQueue = run.then(() => {}, () => {});
  return run;
}

/** Thrown when the operator cancels a confirm dialog. */
export class ConfirmCancelledError extends Error {
  constructor() {
    super('confirm cancelled');
    this.name = 'ConfirmCancelledError';
  }
}
