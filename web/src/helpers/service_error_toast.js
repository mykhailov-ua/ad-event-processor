import { mapServiceError } from './service_error.js';
import { pushToastMessage } from './toast_ui.js';

/**
 * Surfaces API errors per mapServiceError (toast / conflict / retry / inline).
 * Page-level errors should use renderErrorBlock instead.
 *
 * @param {unknown} error
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
