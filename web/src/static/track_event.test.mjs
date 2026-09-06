import assert from 'node:assert/strict';
import test from 'node:test';

import { trackEvent } from './track.js';

function withMockWindow(run, { search = '?msclkid=ms-1&ob_click_id=ob-9' } = {}) {
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
    return run(() => capturedBody);
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

test('trackEvent maps core fields and query attribution', () => {
  withMockWindow((readBody) => {
    void trackEvent({
      endpoint: 'https://track.example/track',
      campaignId: 'camp-1',
      type: 'conversion',
      clickId: 'click-1',
      userId: 'user-1',
      eventId: 'evt-1',
      subs: { sub1: 'a', sub30: 'z' },
    });

    const body = readBody();
    assert.equal(body.campaign_id, 'camp-1');
    assert.equal(body.type, 'conversion');
    assert.equal(body.event_id, 'evt-1');
    assert.equal(body.click_id, 'click-1');
    assert.equal(body.user_id, 'user-1');
    assert.equal(body.sub1, 'a');
    assert.equal(body.sub30, 'z');
    assert.equal(body.msclkid, 'ms-1');
    assert.equal(body.ob_click_id, 'ob-9');
    assert.equal(body.sub2, undefined);
  });
});

test('trackEvent maps full query attribution and obclid alias', () => {
  withMockWindow(
    (readBody) => {
      void trackEvent({
        endpoint: 'https://track.example/track',
        campaignId: 'camp-attrib',
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

test('trackEvent maps subs sub1 through sub30', () => {
  withMockWindow((readBody) => {
    const subs = {};
    for (let index = 1; index <= 30; index += 1) {
      subs[`sub${index}`] = `value-${index}`;
    }

    void trackEvent({
      endpoint: 'https://track.example/track',
      campaignId: 'camp-subs',
      type: 'click',
      eventId: 'evt-subs',
      subs,
    });

    const body = readBody();
    assert.equal(body.event_id, 'evt-subs');
    for (let index = 1; index <= 30; index += 1) {
      assert.equal(body[`sub${index}`], `value-${index}`);
    }
  });
});

test('trackEvent auto event_id and telemetry snapshots', () => {
  withMockWindow((readBody) => {
    globalThis.trackTelemetrySnapshot = () => ({ events: [{ kind: 'telemetry' }] });
    globalThis.trackBiometricsSnapshot = () => ({ events: [{ kind: 'bio' }] });

    void trackEvent({
      endpoint: 'https://track.example/track',
      campaignId: 'camp-2',
      type: 'impression',
    });

    const body = readBody();
    assert.equal(body.event_id, '11111111-1111-4111-8111-111111111111');
    assert.deepEqual(body.telemetry, {
      events: [{ kind: 'telemetry' }, { kind: 'bio' }],
    });
  });
});
