import { maskLevel } from './permissions.js';

export function canShowReportFinancials(permissions: string[]): boolean {
  return maskLevel(permissions) === 'full';
}
