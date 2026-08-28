export function can(permissions: string[] | null | undefined, perm: string): boolean {
  return Array.isArray(permissions) && permissions.includes(perm);
}

export function isTenantUser(role: string | null | undefined): boolean {
  return role === 'U';
}

export function isBuyer(role: string | null | undefined): boolean {
  return role === 'B';
}

export function isTeamLead(role: string | null | undefined): boolean {
  return role === 'TL';
}

export function isMediaBuyer(role: string | null | undefined): boolean {
  return role === 'MB';
}

export function isPublisher(role: string | null | undefined): boolean {
  return role === 'P';
}

export function isPublisherPortal(
  permissions: string[] | null | undefined,
  role: string | null | undefined
): boolean {
  if (isPublisher(role)) return true;
  const perms = permissions ?? [];
  return (
    perms.includes('supply:read:scoped') &&
    !perms.includes('campaigns:read') &&
    !perms.includes('campaigns:read:masked')
  );
}

export function isBillingReadOnly(
  permissions: string[] | null | undefined,
  role: string | null | undefined
): boolean {
  if (isMediaBuyer(role)) return true;
  const perms = permissions ?? [];
  return perms.includes('billing:read') && !perms.includes('billing:write');
}

export function isSupport(role: string | null | undefined): boolean {
  return role === 'S';
}

export type MaskLevelValue = 'full' | 'masked' | 'none';

export function maskLevel(permissions: string[] | null | undefined): MaskLevelValue {
  if (can(permissions, 'campaigns:read')) return 'full';
  if (can(permissions, 'campaigns:read:masked')) return 'masked';
  return 'none';
}
