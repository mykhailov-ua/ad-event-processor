export function can(permissions: string[] | null | undefined, perm: string): boolean {
  return Array.isArray(permissions) && permissions.includes(perm);
}

export function isTenantUser(role: string | null | undefined): boolean {
  return role === 'U';
}

export function isBuyerBoundUser(role: string | null | undefined): boolean {
  return role === 'B' || role === 'BUYER' || role === 'MB' || role === 'MEDIA_BUYER';
}

export function canReadCampaigns(permissions: string[] | null | undefined): boolean {
  return can(permissions, 'campaigns:read') || can(permissions, 'campaigns:read:masked');
}

export function canAccessSelfServe(
  role: string | null | undefined,
  permissions: string[] | null | undefined
): boolean {
  if (isBuyerBoundUser(role)) return true;
  return can(permissions, 'campaigns:read:masked');
}

export function canAccessPublisher(permissions: string[] | null | undefined): boolean {
  return can(permissions, 'supply:read:scoped');
}
