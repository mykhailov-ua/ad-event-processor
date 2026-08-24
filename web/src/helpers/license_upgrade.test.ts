import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  formatLicenseRpsCap,
  resolveLicenseUpgradeHref,
  showLicenseUpgradePath,
} from './license_upgrade.js';

describe('license_upgrade helpers', () => {
  it('shows upgrade path for pilot and unconfigured', () => {
    assert.equal(showLicenseUpgradePath('pilot', 'ACTIVE'), true);
    assert.equal(showLicenseUpgradePath(undefined, 'UNCONFIGURED'), true);
    assert.equal(showLicenseUpgradePath('starter', 'ACTIVE'), false);
  });

  it('formats RPS cap without unlimited wording', () => {
    assert.equal(formatLicenseRpsCap(5000), '5,000 max RPS');
    assert.equal(formatLicenseRpsCap(0), 'see license JWT');
  });

  it('resolves support URL for upgrade CTA', () => {
    const ext = resolveLicenseUpgradeHref('https://t.me/vendor');
    assert.equal(ext.external, true);
    assert.equal(ext.href, 'https://t.me/vendor');
  });
});
