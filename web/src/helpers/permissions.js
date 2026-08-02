/**
 * @param {string[]} permissions
 * @param {string} perm
 * @returns {boolean}
 */
export function can(permissions, perm) {
  return Array.isArray(permissions) && permissions.includes(perm);
}

/**
 * @param {string} role
 * @returns {boolean}
 */
export function isTenantUser(role) {
  return role === 'U';
}

/**
 * @param {string} role
 * @returns {boolean}
 */
export function isBuyer(role) {
  return role === 'B';
}

/**
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
 * @param {string[]} permissions
 * @returns {MaskLevelValue}
 */
export function maskLevel(permissions) {
  if (can(permissions, 'campaigns:read')) return 'full';
  if (can(permissions, 'campaigns:read:masked')) return 'masked';
  return 'none';
}
