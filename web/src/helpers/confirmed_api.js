import { api, isMutation } from './api_client.js';
import { getConfirmLevel } from './confirm_registry.js';
import { requestConfirm, ConfirmCancelledError } from './confirm_ui.js';

/**
 * Call the API after running the confirm-registry gate for mutations.
 *
 * @param {string} path
 * @param {import('./api_client.js').ApiInit} [init]
 * @returns {Promise<{data: any, status: number, headers: Headers}>}
 * @throws {ConfirmCancelledError} when the operator cancels confirmation
 */
export async function apiConfirmed(path, init = {}) {
  const method = (init.method || 'GET').toUpperCase();
  if (isMutation(method)) {
    const registryPath = path.replace(/^\/api\/v1/, '');
    const entry = getConfirmLevel(method, registryPath);
    if (entry.level !== 'none') {
      const ok = await requestConfirm({
        entry,
        method,
        path,
        title: entry.label,
      });
      if (!ok) throw new ConfirmCancelledError();
    }
  }
  return api(path, init);
}
