export const PORTAL_PERMISSIONS = {
  selfserve: 'campaigns:read',
  publisher: 'supply:read:scoped',
  telegram: 'campaigns:read',
  reportSchedules: 'campaigns:read',
  savedViews: 'campaigns:read',
  forecast: 'campaigns:read',
} as const;

export type PortalKey = keyof typeof PORTAL_PERMISSIONS;

export function hasPortalPermission(
  permissions: string[] | undefined,
  key: PortalKey,
): boolean {
  if (permissions === undefined) {
    return true;
  }
  return permissions.includes(PORTAL_PERMISSIONS[key]);
}

export function hasAnyPortalAccess(permissions: string[] | undefined): boolean {
  if (permissions === undefined) {
    return true;
  }
  return Object.values(PORTAL_PERMISSIONS).some((permission) =>
    permissions.includes(permission),
  );
}
