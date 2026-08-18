import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  buildLicenseBannerParts,
  isPilotConvertNudge,
  PILOT_CONVERT_NUDGE_DAYS,
  resolvePilotConvertCTA,
  shouldShowLicenseBanner,
} from './license_banner.js';

describe('license_banner helpers', () => {
  it('shows pilot convert nudge within configured days', () => {
    const license = {
      state: 'active',
      plan_code: 'pilot',
      renew_days: PILOT_CONVERT_NUDGE_DAYS,
      valid_until: '2026-08-23T00:00:00Z',
    };
    assert.equal(isPilotConvertNudge(license), true);
    assert.equal(shouldShowLicenseBanner(license), true);
    const parts = buildLicenseBannerParts(license);
    assert.ok(parts.some((p) => p.includes('Pilot ends in')));
  });

  it('hides healthy starter license with renew window > 7d', () => {
    const license = { state: 'active', plan_code: 'starter', renew_days: 20 };
    assert.equal(shouldShowLicenseBanner(license), false);
  });

  it('shows generic renew banner for pilot at 6d without convert CTA flag', () => {
    const license = { state: 'active', plan_code: 'pilot', renew_days: 6 };
    assert.equal(isPilotConvertNudge(license), false);
    assert.equal(shouldShowLicenseBanner(license), true);
  });

  it('resolves external support URL for pilot CTA', () => {
    const cta = resolvePilotConvertCTA('https://t.me/bidshard_support');
    assert.equal(cta.href, 'https://t.me/bidshard_support');
    assert.equal(cta.external, true);
    assert.equal(cta.label, 'Contact vendor');
  });

  it('falls back to license settings without support URL', () => {
    const cta = resolvePilotConvertCTA('');
    assert.equal(cta.href, '/settings/license');
    assert.equal(cta.external, false);
    assert.equal(cta.label, 'Upgrade license');
  });
});
