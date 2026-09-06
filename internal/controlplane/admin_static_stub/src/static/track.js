'use strict';
(() => {
  function s(n) {
    let e = new URLSearchParams(window.location.search);
    for (let c of ['fbclid', 'gclid', 'ttclid', 'msclkid', 'tblci']) {
      let t = e.get(c);
      t && (n[c] = t);
    }
    let i = e.get('ob_click_id') || e.get('obclid');
    i && (n.ob_click_id = i);
  }
  function a(n) {
    let e = [],
      i = globalThis.trackTelemetrySnapshot;
    if (typeof i == 'function') {
      let t = i();
      t && t.events && t.events.length && (e = e.concat(t.events));
    }
    let c = globalThis.trackBiometricsSnapshot;
    if (typeof c == 'function') {
      let t = c();
      t && t.events && t.events.length && (e = e.concat(t.events));
    }
    e.length && (n.telemetry = { events: e });
  }
  function r(n) {
    let e = { campaign_id: n.campaignId, type: n.type },
      i =
        n.eventId ||
        (typeof crypto != 'undefined' && typeof crypto.randomUUID == 'function'
          ? crypto.randomUUID()
          : '');
    i && (e.event_id = i),
      n.clickId && (e.click_id = n.clickId),
      n.userId && (e.user_id = n.userId);
    let c = n.subs || {};
    for (let t = 1; t <= 30; t += 1) {
      let o = `sub${t}`;
      c[o] && (e[o] = c[o]);
    }
    return (
      s(e),
      a(e),
      fetch(n.endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(e),
        keepalive: !0,
        credentials: 'omit',
      })
    );
  }
  globalThis.trackEvent = r;
})();
