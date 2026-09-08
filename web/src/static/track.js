/**
 * POST /track from a landing page (browser pixel). Requires TRACK_CORS_ORIGINS on tracker.
 *
 * Canonical source for tracker go:embed and admin dist copy.
 * Build: node web/scripts/build_track_pixel.mjs
 *
 * Verify:
 * node web/scripts/build_track_pixel.mjs --check
 * node --import ./scripts/test_aliases.mjs --test --experimental-strip-types src/static/track_event.test.mjs
 * go test ./internal/track/ -short -run TestTrackPixelContract -count=1
 */

function appendQueryAttribution(body) {
  const params = new URLSearchParams(window.location.search);
  for (const key of ['fbclid', 'gclid', 'ttclid', 'msclkid', 'tblci']) {
    const value = params.get(key);
    if (value) {
      body[key] = value;
    }
  }
  const obClickId = params.get('ob_click_id') || params.get('obclid');
  if (obClickId) {
    body.ob_click_id = obClickId;
  }
}

async function appendTelemetry(body, campaignId) {
  let events = [];
  const telemetrySnapshot = globalThis.trackTelemetrySnapshot;
  if (typeof telemetrySnapshot === 'function') {
    const snapshot = telemetrySnapshot();
    if (snapshot && snapshot.events && snapshot.events.length) {
      events = events.concat(snapshot.events);
    }
  }
  const biometricsSnapshot = globalThis.trackBiometricsSnapshot;
  if (typeof biometricsSnapshot === 'function') {
    const snapshot = biometricsSnapshot();
    if (snapshot && snapshot.events && snapshot.events.length) {
      events = events.concat(snapshot.events);
    }
  }
  if (events.length) {
    body.telemetry = { events };
  }
  const arm = globalThis.trackAntifraudArm;
  if (typeof arm === 'function' && campaignId) {
    arm(campaignId);
  }
  const whenReady = globalThis.trackAntifraudWhenReady;
  if (typeof whenReady === 'function') {
    await whenReady();
  }
  const antifraudSnapshot = globalThis.trackAntifraudSnapshot;
  if (typeof antifraudSnapshot === 'function') {
    const snapshot = antifraudSnapshot();
    if (snapshot) {
      body.antifraud = snapshot;
    }
  }
}

export async function trackEvent(opts) {
  const body = {
    campaign_id: opts.campaignId,
    type: opts.type,
  };
  const eventId =
    opts.eventId ||
    (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
      ? crypto.randomUUID()
      : '');
  if (eventId) {
    body.event_id = eventId;
  }
  if (opts.clickId) {
    body.click_id = opts.clickId;
  }
  if (opts.userId) {
    body.user_id = opts.userId;
  }
  const subs = opts.subs || {};
  for (let index = 1; index <= 30; index += 1) {
    const key = `sub${index}`;
    if (subs[key]) {
      body[key] = subs[key];
    }
  }
  appendQueryAttribution(body);
  await appendTelemetry(body, opts.campaignId);
  return fetch(opts.endpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
    keepalive: true,
    credentials: 'omit',
  });
}

globalThis.trackEvent = trackEvent;
