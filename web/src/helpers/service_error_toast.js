import { mapServiceError } from './service_error.js';
import { pushToastMessage } from './toast_ui.js';

/**
 * Surface a service error as a toast when appropriate for transient failures.
 *
 * @param {unknown} error
 * @returns {void}
 */
export function surfaceServiceErrorToast(error) {
  if (!error) return;
  const view = mapServiceError(error);
  if (
    view.kind === 'toast'
    || view.kind === 'conflict'
    || view.kind === 'retry'
    || view.kind === 'inline'
  ) {
    pushToastMessage({
      title: view.title,
      message: view.message,
      code: view.code,
    });
  }
}
