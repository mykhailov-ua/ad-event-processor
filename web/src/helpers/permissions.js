/**
 * Test whether the permission list includes a specific capability.
 *
 * @param {string[]} permissions
 * @param {string} perm
 * @returns {boolean}
 */
export function can(permissions, perm) {
  return Array.isArray(permissions) && permissions.includes(perm);
}

/**
 * Test whether the role is a tenant-scoped user.
 *
 * @param {string} role
 * @returns {boolean}
 */
export function isTenantUser(role) {
  return role === 'U';
}

/**
 * Test whether the role is a buyer.
 *
 * @param {string} role
 * @returns {boolean}
 */
export function isBuyer(role) {
  return role === 'B';
}

/**
 * Test whether the role is support staff.
 *
 * @param {string} role
 * @returns {boolean}
 */
export function isSupport(role) {
  return role === 'S';
}

/**
 * @typedef {'full'|'masked'|'none'} MaskLevelValue
 */

/**
 * Resolve campaign read masking level from permissions.
 *
 * @param {string[]} permissions
 * @returns {MaskLevelValue}
 */
export function maskLevel(permissions) {
  if (can(permissions, 'campaigns:read')) return 'full';
  if (can(permissions, 'campaigns:read:masked')) return 'masked';
  return 'none';
}
