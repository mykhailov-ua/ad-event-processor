import { maskLevel } from './permissions.js';

/**
 * Test whether financial report columns may be shown for the permission set.
 *
 * @param {string[]} permissions
 * @returns {boolean}
 */
export function canShowReportFinancials(permissions) {
  return maskLevel(permissions) === 'full';
}
