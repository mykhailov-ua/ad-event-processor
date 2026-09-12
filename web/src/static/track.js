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
  const evSnapshot = globalThis.tagEvSnapshot;
  if (typeof evSnapshot === 'function') {
    const snapshot = evSnapshot();
    if (snapshot && snapshot.events && snapshot.events.length) {
      events = events.concat(snapshot.events);
    }
  }
  const inSnapshot = globalThis.tagInSnapshot;
  if (typeof inSnapshot === 'function') {
    const snapshot = inSnapshot();
    if (snapshot && snapshot.events && snapshot.events.length) {
      events = events.concat(snapshot.events);
    }
  }
  if (events.length) {
    body.ev = { events };
  }
  const arm = globalThis.tagCtxArm;
  if (typeof arm === 'function' && campaignId) {
    arm(campaignId);
  }
  const whenReady = globalThis.tagCtxReady;
  if (typeof whenReady === 'function') {
    await whenReady();
  }
  const ctxSnapshot = globalThis.tagCtxSnapshot;
  if (typeof ctxSnapshot === 'function') {
    const snapshot = ctxSnapshot();
    if (snapshot) {
      body.ctx = snapshot;
    }
  }
}

async function waitMinDwellMs(ms) {
  const dwell = Number(ms);
  if (!dwell || dwell <= 0) {
    return;
  }
  const start = performance.now();
  while (performance.now() - start < dwell) {
    await new Promise((resolve) => setTimeout(resolve, Math.min(dwell, 32)));
  }
}

export async function sendEvent(opts) {
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
  if (opts.payoutMicro != null && opts.payoutMicro !== '') {
    body.payout_micro = String(opts.payoutMicro);
  }
  if (opts.goal) {
    body.goal_name = opts.goal;
  }
  appendQueryAttribution(body);
  await appendTelemetry(body, opts.campaignId);
  await waitMinDwellMs(opts.minDwellMs);
  return fetch(opts.endpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
    keepalive: true,
    credentials: 'omit',
  });
}

const sdkState = {
  campaignId: '',
  endpoint: '',
  clickId: '',
  subs: {},
};

export function init(campaignId, opts = {}) {
  sdkState.campaignId = String(campaignId || '');
  if (opts.endpoint) {
    sdkState.endpoint = opts.endpoint;
  }
  if (opts.clickId) {
    sdkState.clickId = opts.clickId;
  }
  if (opts.subs) {
    sdkState.subs = { ...opts.subs };
  }
}

export async function click(opts = {}) {
  return sendEvent({
    campaignId: opts.campaignId || sdkState.campaignId,
    type: 'click',
    endpoint: opts.endpoint || sdkState.endpoint,
    clickId: opts.clickId || sdkState.clickId,
    subs: { ...sdkState.subs, ...(opts.subs || {}) },
    minDwellMs: opts.minDwellMs,
  });
}

export async function conversion(goal, payoutMicro, opts = {}) {
  const subs = { ...sdkState.subs, ...(opts.subs || {}) };
  if (goal) {
    subs.sub1 = goal;
  }
  return sendEvent({
    campaignId: opts.campaignId || sdkState.campaignId,
    type: 'conversion',
    endpoint: opts.endpoint || sdkState.endpoint,
    clickId: opts.clickId || sdkState.clickId,
    eventId: opts.eventId,
    goal,
    payoutMicro,
    subs,
    minDwellMs: opts.minDwellMs,
  });
}

function bootFromScriptTag() {
  const tag = document.currentScript;
  if (!tag) {
    return;
  }
  const campaignId = tag.getAttribute('data-campaign-id');
  const endpoint = tag.getAttribute('data-track-endpoint');
  if (!campaignId || !endpoint) {
    return;
  }
  init(campaignId, {
    endpoint,
    clickId: tag.getAttribute('data-click-id') || '',
  });
  if (tag.getAttribute('data-auto-conversion') === '1') {
    conversion(
      tag.getAttribute('data-goal') || 'conversion',
      tag.getAttribute('data-payout-micro') || ''
    );
  }
}

bootFromScriptTag();

globalThis.sendEvent = sendEvent;
globalThis.init = init;
globalThis.click = click;
globalThis.conversion = conversion;
