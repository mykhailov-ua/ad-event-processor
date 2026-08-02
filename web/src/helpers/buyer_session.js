import { isBuyer, isTenantUser } from './permissions.js';

/**
 * Test whether the session role is bound to a single customer account.
 *
 * @param {string} [role]
 * @returns {boolean}
 */
export function hasBoundCustomer(role) {
  return isTenantUser(role) || isBuyer(role);
}

/**
 * Return the customer id from the authenticated user when session-scoped.
 *
 * @param {{ role?: string, customer_id?: string }|null|undefined} user
 * @returns {string}
 */
export function boundCustomerId(user) {
  if (!user || !hasBoundCustomer(user.role)) return '';
  return user.customer_id ?? '';
}
