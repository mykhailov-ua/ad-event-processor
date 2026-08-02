import { describe, it, expect } from 'vitest';
import { can, maskLevel, isTenantUser } from './permissions.js';

describe('permissions.js', () => {
  it('checks permission correctly', () => {
    expect(can(['campaigns:read', 'customers:read'], 'campaigns:read')).toBe(true);
    expect(can(['customers:read'], 'campaigns:read')).toBe(false);
    expect(can(null, 'campaigns:read')).toBe(false);
  });

  it('determines mask level for buyer vs admin', () => {
    expect(maskLevel(['campaigns:read'])).toBe('full');
    expect(maskLevel(['campaigns:read:masked'])).toBe('masked');
    expect(maskLevel(['customers:read'])).toBe('none');
  });

  it('identifies tenant user role U', () => {
    expect(isTenantUser('U')).toBe(true);
    expect(isTenantUser('A')).toBe(false);
  });
});
