import assert from 'node:assert/strict';
import test from 'node:test';

import { seedDeterministicUuid } from '../lib/uuid.ts';
import { trackEvent } from './track.js';

const CAMPAIGN_1 = seedDeterministicUuid('campaign', 1);
const CAMPAIGN_2 = seedDeterministicUuid('campaign', 2);
const CAMPAIGN_DWELL = seedDeterministicUuid('campaign', 3);
const CAMPAIGN_ATTRIB = seedDeterministicUuid('campaign', 4);
const CAMPAIGN_SUBS = seedDeterministicUuid('campaign', 5);
const USER_1 = seedDeterministicUuid('user', 1);
const EVENT_1 = seedDeterministicUuid('event', 1);
const EVENT_SUBS = seedDeterministicUuid('event', 2);

async function withMockWindow(run, { search = '?msclkid=ms-1&ob_click_id=ob-9' } = {}) {
  const previousWindow = globalThis.window;
  const previousFetch = globalThis.fetch;
  const previousCrypto = globalThis.crypto;

  globalThis.window = {
    location: { search },
  };
  Object.defineProperty(globalThis, 'crypto', {
    configurable: true,
    value: {
      randomUUID: () => '11111111-1111-4111-8111-111111111111',
    },
  });

  let capturedBody;
  globalThis.fetch = async (_url, init) => {
    capturedBody = JSON.parse(String(init.body));
    return { ok: true };
  };

  try {
    return await run(() => capturedBody);
  } finally {
    globalThis.window = previousWindow;
    globalThis.fetch = previousFetch;
    Object.defineProperty(globalThis, 'crypto', {
      configurable: true,
      value: previousCrypto,
    });
    delete globalThis.trackTelemetrySnapshot;
    delete globalThis.trackBiometricsSnapshot;
  }
}

test('trackEvent honors minDwellMs before fetch', async () => {
  const previousWindow = globalThis.window;
  const previousFetch = globalThis.fetch;
  let capturedBody;
  globalThis.window = { location: { search: '' } };
  globalThis.fetch = async (_url, init) => {
    capturedBody = JSON.parse(String(init.body));
    return { ok: true };
  };
  try {
    const started = performance.now();
    await trackEvent({
      endpoint: 'https://track.example/track',
      campaignId: CAMPAIGN_DWELL,
      type: 'conversion',
      minDwellMs: 35,
    });
    assert.ok(performance.now() - started >= 30);
    assert.equal(capturedBody.campaign_id, CAMPAIGN_DWELL);
  } finally {
    globalThis.window = previousWindow;
    globalThis.fetch = previousFetch;
  }
});

test('trackEvent maps core fields and query attribution', async () => {
  await withMockWindow(async (readBody) => {
    await trackEvent({
      endpoint: 'https://track.example/track',
      campaignId: CAMPAIGN_1,
      type: 'conversion',
      clickId: 'click-1',
      userId: USER_1,
      eventId: EVENT_1,
      subs: { sub1: 'a', sub30: 'z' },
    });

    const body = readBody();
    assert.equal(body.campaign_id, CAMPAIGN_1);
    assert.equal(body.type, 'conversion');
    assert.equal(body.event_id, EVENT_1);
    assert.equal(body.click_id, 'click-1');
    assert.equal(body.user_id, USER_1);
    assert.equal(body.sub1, 'a');
    assert.equal(body.sub30, 'z');
    assert.equal(body.msclkid, 'ms-1');
    assert.equal(body.ob_click_id, 'ob-9');
    assert.equal(body.sub2, undefined);
  });
});

test('trackEvent maps full query attribution and obclid alias', async () => {
  await withMockWindow(
    async (readBody) => {
      await trackEvent({
        endpoint: 'https://track.example/track',
        campaignId: CAMPAIGN_ATTRIB,
        type: 'click',
      });

      const body = readBody();
      assert.equal(body.fbclid, 'fb-1');
      assert.equal(body.gclid, 'gc-1');
      assert.equal(body.ttclid, 'tt-1');
      assert.equal(body.msclkid, 'ms-2');
      assert.equal(body.tblci, 'tb-1');
      assert.equal(body.ob_click_id, 'ob-alias');
    },
    {
      search: '?fbclid=fb-1&gclid=gc-1&ttclid=tt-1&msclkid=ms-2&tblci=tb-1&obclid=ob-alias',
    }
  );
});

test('trackEvent maps subs sub1 through sub30', async () => {
  await withMockWindow(async (readBody) => {
    const subs = {};
    for (let index = 1; index <= 30; index += 1) {
      subs[`sub${index}`] = `value-${index}`;
    }

    await trackEvent({
      endpoint: 'https://track.example/track',
      campaignId: CAMPAIGN_SUBS,
      type: 'click',
      eventId: EVENT_SUBS,
      subs,
    });

    const body = readBody();
    assert.equal(body.event_id, EVENT_SUBS);
    for (let index = 1; index <= 30; index += 1) {
      assert.equal(body[`sub${index}`], `value-${index}`);
    }
  });
});

test('trackEvent auto event_id and telemetry snapshots', async () => {
  await withMockWindow(async (readBody) => {
    globalThis.trackTelemetrySnapshot = () => ({ events: [{ kind: 'telemetry' }] });
    globalThis.trackBiometricsSnapshot = () => ({ events: [{ kind: 'bio' }] });

    await trackEvent({
      endpoint: 'https://track.example/track',
      campaignId: CAMPAIGN_2,
      type: 'impression',
    });

    const body = readBody();
    assert.equal(body.event_id, '11111111-1111-4111-8111-111111111111');
    assert.deepEqual(body.telemetry, {
      events: [{ kind: 'telemetry' }, { kind: 'bio' }],
    });
  });
});
