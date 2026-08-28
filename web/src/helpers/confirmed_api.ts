import { api, isMutation, type ApiInit, type ApiResult } from './api_client.js';
import { getConfirmLevel } from './confirm_registry.js';
import { requestConfirm, ConfirmCancelledError } from './confirm_ui.js';

export async function apiConfirmed<T = unknown>(
  path: string,
  init: ApiInit = {}
): Promise<ApiResult<T>> {
  const method = (init.method || 'GET').toUpperCase();
  if (isMutation(method)) {
    const pathWithoutQuery = path.split('?')[0] ?? path;
    const registryPath = pathWithoutQuery.replace(/^\/api\/v1/, '');
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
  return api<T>(path, init);
}

export { ConfirmCancelledError };
