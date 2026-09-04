import {
  normalizeDevMockRole,
  writeDevMockRoleToStorage,
  type DevMockRoleId,
} from '@/api/dev_mock/rbac';

/** Call once before the React tree mounts (main.tsx). */
export function initDevMockRoleFromUrl(): void {
  if (typeof window === 'undefined') {
    return;
  }
  const params = new URLSearchParams(window.location.search);
  const role = params.get('admin_dev_role');
  if (!role) {
    return;
  }
  writeDevMockRoleToStorage(normalizeDevMockRole(role));
  params.delete('admin_dev_role');
  const nextQuery = params.toString();
  const nextUrl = `${window.location.pathname}${nextQuery ? `?${nextQuery}` : ''}${window.location.hash}`;
  window.history.replaceState(null, '', nextUrl);
}

export function setDevMockRole(role: DevMockRoleId): void {
  writeDevMockRoleToStorage(role);
}
