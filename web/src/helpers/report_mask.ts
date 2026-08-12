import { maskLevel } from './permissions.js';

/**
 * Test whether financial report columns may be shown for the permission set.
 */
export function canShowReportFinancials(permissions: string[]): boolean {
  return maskLevel(permissions) === 'full';
}
