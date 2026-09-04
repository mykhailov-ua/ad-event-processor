import type { MockResult } from './handler_types.ts';

export type DevMockRoleId = 'A' | 'M' | 'U' | 'B' | 'TL' | 'MB' | 'S' | 'P';

const ROLE_PERMISSIONS: Record<DevMockRoleId, string[]> = {
  A: [
    '*',
    'customers:write',
    'customers:read',
    'campaigns:write',
    'campaigns:read',
    'brands:write',
    'brands:read',
    'settings:write',
    'settings:read',
    'blacklist:write',
    'blacklist:read',
    'audit:read',
    'users:write',
    'shards:write',
    'shards:read',
    'ops:write',
    'rtb:read',
    'rtb:write',
    'billing:read',
  ],
  M: [
    'customers:write',
    'customers:read',
    'campaigns:write',
    'campaigns:read',
    'brands:write',
    'brands:read',
    'audit:read',
  ],
  U: ['campaigns:write', 'campaigns:read', 'customers:read', 'brands:write', 'brands:read'],
  B: ['campaigns:read:masked', 'campaigns:pause'],
  TL: ['campaigns:read', 'campaigns:write', 'campaigns:pause', 'customers:read', 'billing:read'],
  MB: ['campaigns:read', 'campaigns:write', 'customers:read'],
  S: ['campaigns:read:masked', 'audit:read'],
  P: ['supply:read:scoped', 'customers:read'],
};

const STORAGE_KEY = 'adminDevMockRole';

let testRoleOverride: DevMockRoleId | null = null;

export function normalizeDevMockRole(role: string): DevMockRoleId {
  switch (role.trim().toUpperCase()) {
    case 'ADMIN':
    case 'SA':
    case 'SUPERADMIN':
    case 'A':
      return 'A';
    case 'MANAGER':
    case 'M':
      return 'M';
    case 'CUSTOMER':
    case 'USER':
    case 'C':
    case 'U':
      return 'U';
    case 'BUYER':
    case 'B':
      return 'B';
    case 'TEAM_LEAD':
    case 'TEAMLEAD':
    case 'TL':
      return 'TL';
    case 'MEDIA_BUYER':
    case 'MEDIABUYER':
    case 'MB':
      return 'MB';
    case 'SUPPORT':
    case 'S':
      return 'S';
    case 'PUBLISHER':
    case 'P':
      return 'P';
    default:
      return 'A';
  }
}

export function readDevMockRoleFromStorage(): DevMockRoleId | null {
  if (typeof window === 'undefined') {
    return null;
  }
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    if (!stored) {
      return null;
    }
    return normalizeDevMockRole(stored);
  } catch {
    return null;
  }
}

export function writeDevMockRoleToStorage(role: DevMockRoleId): void {
  if (typeof window === 'undefined') {
    return;
  }
  try {
    window.localStorage.setItem(STORAGE_KEY, role);
  } catch {
    // Private browsing or blocked storage.
  }
}

export function getDevMockRole(): DevMockRoleId {
  if (testRoleOverride) {
    return testRoleOverride;
  }
  return readDevMockRoleFromStorage() ?? 'A';
}

export function devMockSessionRoleLabel(role: DevMockRoleId): string {
  if (role === 'A') {
    return 'admin';
  }
  return role;
}

export function devMockPermissionsForRole(role: DevMockRoleId): string[] {
  return ROLE_PERMISSIONS[role];
}

export function devMockCurrentPermissions(): string[] {
  return devMockPermissionsForRole(getDevMockRole());
}

export function devMockHasPermission(
  permission: string,
  permissions: string[] = devMockCurrentPermissions(),
): boolean {
  return permissions.some((entry) => entry === '*' || entry === permission);
}

export function devMockHasAnyPermission(
  required: string[],
  permissions: string[] = devMockCurrentPermissions(),
): boolean {
  return required.some((permission) => devMockHasPermission(permission, permissions));
}

export function devMockForbiddenResult(): MockResult {
  return {
    status: 403,
    body: { error: { code: 'FORBIDDEN', message: 'forbidden: insufficient permissions' } },
    contentType: 'application/json',
  };
}

function requiredPermissions(pathname: string, method: string): string[] | null {
  if (pathname.startsWith('/api/v1/ops/')) {
    if (method === 'GET') {
      return ['shards:read'];
    }
    if (pathname === '/api/v1/ops/roles/reload') {
      return ['settings:write'];
    }
    if (pathname.startsWith('/api/v1/ops/blacklist')) {
      return ['blacklist:write', 'blacklist:read'];
    }
    return ['shards:write', 'ops:write', 'blacklist:write', 'settings:write'];
  }
  if (pathname.startsWith('/api/v1/settings/')) {
    return method === 'GET' ? ['settings:read'] : ['settings:write'];
  }
  if (pathname.startsWith('/api/v1/audit')) {
    return ['audit:read'];
  }
  if (pathname.startsWith('/api/v1/rtb/')) {
    return method === 'GET' ? ['rtb:read'] : ['rtb:write'];
  }
  if (pathname.startsWith('/api/v1/dashboards/buyer')) {
    return ['campaigns:read', 'campaigns:read:masked'];
  }
  if (pathname.startsWith('/api/v1/dashboards/fraud')) {
    return ['audit:read'];
  }
  if (
    pathname.startsWith('/api/v1/dashboards/operator') ||
    pathname.startsWith('/api/v1/dashboards/adops')
  ) {
    return ['shards:read'];
  }
  return null;
}

export function devMockAuthorizeRequest(pathname: string, method: string): MockResult | null {
  const required = requiredPermissions(pathname, method);
  if (!required) {
    return null;
  }
  if (!devMockHasAnyPermission(required)) {
    return devMockForbiddenResult();
  }
  return null;
}

export function setDevMockRoleForTests(role: DevMockRoleId | string | null): void {
  testRoleOverride = role ? normalizeDevMockRole(role) : null;
}

export function resetDevMockRoleForTests(): void {
  testRoleOverride = null;
}
