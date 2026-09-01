import type { MetaResponse } from '@/api/types';

const BLOCKED_LICENSE_STATES = new Set(['EXPIRED', 'REVOKED']);

export function readBootstrapComplete(meta: MetaResponse | undefined): boolean {
  return meta?.bootstrap_complete === true;
}

export function licenseNeedsSetup(meta: MetaResponse | undefined): boolean {
  const state = meta?.license?.state?.trim();
  if (!state) {
    return true;
  }
  return BLOCKED_LICENSE_STATES.has(state);
}

export function licenseStateLabel(meta: MetaResponse | undefined): string {
  const state = meta?.license?.state?.trim();
  return state || 'missing';
}
