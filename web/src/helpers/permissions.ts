/**
 * Test whether the permission list includes a specific capability.
 */
export function can(permissions: string[] | null | undefined, perm: string): boolean {
  return Array.isArray(permissions) && permissions.includes(perm);
}

/**
 * Test whether the role is a tenant-scoped user.
 */
export function isTenantUser(role: string | null | undefined): boolean {
  return role === 'U';
}

/**
 * Test whether the role is a buyer (masked portfolio).
 */
export function isBuyer(role: string | null | undefined): boolean {
  return role === 'B';
}

/**
 * Test whether the role is a team lead.
 */
export function isTeamLead(role: string | null | undefined): boolean {
  return role === 'TL';
}

/**
 * Test whether the role is a media buyer (scoped campaign owner).
 */
export function isMediaBuyer(role: string | null | undefined): boolean {
  return role === 'MB';
}

/**
 * Test whether billing views should be read-only (no top-up / exports).
 */
export function isBillingReadOnly(
  permissions: string[] | null | undefined,
  role: string | null | undefined,
): boolean {
  if (isMediaBuyer(role)) return true;
  const perms = permissions ?? [];
  return perms.includes('billing:read') && !perms.includes('billing:write');
}

/**
 * Test whether the role is support staff.
 */
export function isSupport(role: string | null | undefined): boolean {
  return role === 'S';
}

export type MaskLevelValue = 'full' | 'masked' | 'none';

/**
 * Resolve campaign read masking level from permissions.
 */
export function maskLevel(permissions: string[] | null | undefined): MaskLevelValue {
  if (can(permissions, 'campaigns:read')) return 'full';
  if (can(permissions, 'campaigns:read:masked')) return 'masked';
  return 'none';
}
