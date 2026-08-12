import { isBuyer, isTenantUser } from './permissions.js';

/**
 * Test whether the session role is bound to a single customer account.
 */
export function hasBoundCustomer(role?: string | null): boolean {
  return isTenantUser(role) || isBuyer(role);
}

/**
 * Return the customer id from the authenticated user when session-scoped.
 */
export function boundCustomerId(
  user: { role?: string; customer_id?: string } | null | undefined,
): string {
  if (!user || !hasBoundCustomer(user.role)) return '';
  return user.customer_id ?? '';
}
