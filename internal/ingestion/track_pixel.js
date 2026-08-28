'use strict';
(() => {
  function a(e) {
    let t = { campaign_id: e.campaignId, type: e.type },
      o = e.eventId || (typeof crypto < 'u' && crypto.randomUUID ? crypto.randomUUID() : '');
    o && (t.event_id = o),
      e.clickId && (t.click_id = e.clickId),
      e.userId && (t.user_id = e.userId);
    let d = e.subs || {};
    for (let c = 1; c <= 30; c += 1) {
      let i = `sub${c}`;
      d[i] && (t[i] = d[i]);
    }
    let n = new URLSearchParams(window.location.search);
    for (let c of ['fbclid', 'gclid', 'ttclid', 'msclkid', 'tblci']) {
      let i = n.get(c);
      i && (t[c] = i);
    }
    let r = n.get('ob_click_id') || n.get('obclid');
    r && (t.ob_click_id = r);
    let events = [];
    let s = globalThis.trackTelemetrySnapshot;
    if (typeof s === 'function') {
      let m = s();
      m && m.events && m.events.length && (events = events.concat(m.events));
    }
    let b = globalThis.trackBiometricsSnapshot;
    if (typeof b === 'function') {
      let bm = b();
      bm && bm.events && bm.events.length && (events = events.concat(bm.events));
    }
    events.length && (t.telemetry = { events: events });
    return fetch(e.endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(t),
      keepalive: !0,
      credentials: 'omit',
    });
  }
  globalThis.trackEvent = a;
})();
