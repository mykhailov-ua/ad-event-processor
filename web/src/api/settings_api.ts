import { apiJson, type ApiRequestInit } from './client.js';
import type {
  PlatformApplyRequest,
  PlatformApplyResponse,
  PlatformBootstrapRequest,
  PlatformSettingsPatch,
  PlatformSettingsView,
} from './types.js';

export async function getPlatformSettings(signal?: AbortSignal): Promise<PlatformSettingsView> {
  return apiJson<PlatformSettingsView>('/api/v1/settings/platform', { signal });
}

export async function patchPlatformSettings(
  patch: PlatformSettingsPatch,
  signal?: AbortSignal,
): Promise<PlatformSettingsView> {
  return apiJson<PlatformSettingsView>('/api/v1/settings/platform', {
    method: 'PATCH',
    body: JSON.stringify(patch),
    signal,
  });
}

export async function bootstrapPlatformSettings(
  installToken: string,
  body: PlatformBootstrapRequest,
  signal?: AbortSignal,
): Promise<PlatformSettingsView> {
  return apiJson<PlatformSettingsView>('/api/v1/settings/platform/bootstrap', {
    method: 'POST',
    body: JSON.stringify(body),
    headers: { 'X-Install-Token': installToken },
    signal,
  });
}

export async function applyPlatformSettings(
  body?: PlatformApplyRequest,
  signal?: AbortSignal,
): Promise<PlatformApplyResponse> {
  const installRoot = body?.install_root?.trim();
  const init: ApiRequestInit = {
    method: 'POST',
    signal,
  };
  if (installRoot) {
    init.body = JSON.stringify({ install_root: installRoot });
  }
  return apiJson<PlatformApplyResponse>('/api/v1/settings/platform/apply', init);
}
