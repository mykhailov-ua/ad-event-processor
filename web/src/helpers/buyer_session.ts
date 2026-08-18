import { isBuyer, isMediaBuyer, isTeamLead, isTenantUser } from './permissions.js';

export function hasBoundCustomer(role?: string | null): boolean {
  return isTenantUser(role) || isBuyer(role) || isTeamLead(role) || isMediaBuyer(role);
}

export function boundCustomerId(
  user: { role?: string; customer_id?: string } | null | undefined
): string {
  if (!user || !hasBoundCustomer(user.role)) return '';
  return user.customer_id ?? '';
}
