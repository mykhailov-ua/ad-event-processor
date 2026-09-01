import assert from 'node:assert/strict';
import test from 'node:test';

import { parsePlatformSettingsSnapshot } from './settings_snapshot.ts';

test('parsePlatformSettingsSnapshot maps platform config fields from API payload', () => {
  const snapshot = parsePlatformSettingsSnapshot({
    bootstrap_complete: true,
    restart_required: ['tracking_domain'],
    click_url_template: 'https://track.example.com/click',
    openrtb_endpoint_template: 'https://track.example.com/openrtb',
    config: {
      tracking_domain: 'track.example.com',
      default_currency: 'USD',
      timezone: 'UTC',
      ingress_schema: 'ad_event_processor_native',
      telemetry_enabled: true,
      profile: 'single_vps',
      edge_xdp: false,
      edge_expose_click: true,
      edge_expose_openrtb: false,
      network_interface: 'eth0',
      stripe: {
        enabled: true,
        checkout_success_url: 'https://example.com/success',
        checkout_cancel_url: 'https://example.com/cancel',
      },
    },
    secrets: {
      stripe_secret_key: '****1234',
      stripe_webhook_secret: '****abcd',
    },
  });

  assert.equal(snapshot.bootstrapComplete, true);
  assert.deepEqual(snapshot.restartPending, ['tracking_domain']);
  assert.equal(snapshot.config.trackingDomain, 'track.example.com');
  assert.equal(snapshot.config.stripeEnabled, true);
  assert.equal(snapshot.secrets.stripeSecretKey, '****1234');
});

test('parsePlatformSettingsSnapshot treats boolean restart_required as pending', () => {
  const snapshot = parsePlatformSettingsSnapshot({
    bootstrap_complete: false,
    restart_required: true,
    config: {},
    secrets: {},
  });

  assert.deepEqual(snapshot.restartPending, ['configuration']);
});

test('parsePlatformSettingsSnapshot returns empty restart list when restart_required is empty', () => {
  const snapshot = parsePlatformSettingsSnapshot({
    restart_required: [],
    config: {},
    secrets: {},
  });

  assert.deepEqual(snapshot.restartPending, []);
});
