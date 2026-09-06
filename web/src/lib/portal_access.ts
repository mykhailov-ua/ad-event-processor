// Client nav gate for optional portal routes (self-serve, publisher, telegram, ...).
// Hides sidebar links when permissions[] is present; server RBAC remains authoritative (control-plane.mdc).
// permissions === undefined means bootstrap not loaded yet: show links (fail-open for nav chrome only).
export const PORTAL_PERMISSIONS = {
  selfserve: 'campaigns:read',
  publisher: 'supply:read:scoped',
  telegram: 'campaigns:read',
  reportSchedules: 'campaigns:read',
  savedViews: 'campaigns:read',
  forecast: 'campaigns:read',
} as const;

export type PortalKey = keyof typeof PORTAL_PERMISSIONS;

export function hasPortalPermission(permissions: string[] | undefined, key: PortalKey): boolean {
  if (permissions === undefined) {
    return true;
  }
  return permissions.includes(PORTAL_PERMISSIONS[key]);
}

export function hasAnyPortalAccess(permissions: string[] | undefined): boolean {
  if (permissions === undefined) {
    return true;
  }
  return Object.values(PORTAL_PERMISSIONS).some((permission) => permissions.includes(permission));
}
