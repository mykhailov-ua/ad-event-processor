import { apiFetch, apiJson, parseApiError } from './client.js';

export type AccessCapability = {
  id: string;
  label: string;
  group: string;
  description?: string;
};

export type AccessPermission = {
  id: string;
  label: string;
  internal?: boolean;
  description?: string;
};

export type AccessCatalogResponse = {
  capabilities: AccessCapability[];
  permissions: AccessPermission[];
};

export type AccessRoleView = {
  code: string;
  scope: string;
  builtin: boolean;
  label?: string;
  capabilities?: string[];
  permissions?: string[];
  compiled_permissions: string[];
  member_count?: number;
};

export type AccessRolesResponse = {
  version: number;
  revision: number;
  roles: Record<string, AccessRoleView>;
};

export type AccessValidateResponse = {
  valid: boolean;
  errors?: { field: string; code: string }[];
  roles?: Record<string, AccessRoleView>;
};

export async function getAccessCatalog(signal?: AbortSignal): Promise<AccessCatalogResponse> {
  return apiJson<AccessCatalogResponse>('/api/v1/access/catalog', { signal });
}

export async function getAccessRoles(
  params?: { scope?: string },
  signal?: AbortSignal
): Promise<AccessRolesResponse> {
  const query = params?.scope ? `?scope=${encodeURIComponent(params.scope)}` : '';
  return apiJson<AccessRolesResponse>(`/api/v1/access/roles${query}`, { signal });
}

export async function getAccessRolesYaml(signal?: AbortSignal): Promise<string> {
  const response = await apiFetch('/api/v1/access/roles.yaml', { signal });
  if (!response.ok) {
    throw await parseApiError(response);
  }
  return response.text();
}

export async function validateAccessRolesYaml(
  yamlBody: string,
  signal?: AbortSignal
): Promise<AccessValidateResponse> {
  return apiJson<AccessValidateResponse>('/api/v1/access/roles/validate', {
    method: 'POST',
    headers: { 'Content-Type': 'application/yaml' },
    body: yamlBody,
    signal,
  });
}

export async function applyAccessRolesYaml(
  yamlBody: string,
  revision: number,
  signal?: AbortSignal
): Promise<{ revision: number }> {
  return apiJson<{ revision: number }>('/api/v1/access/roles.yaml', {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/yaml',
      'If-Match': String(revision),
    },
    body: yamlBody,
    signal,
  });
}
