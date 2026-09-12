'use strict';
(() => {
  function l(n) {
    let t = new URLSearchParams(window.location.search);
    for (let o of ['fbclid', 'gclid', 'ttclid', 'msclkid', 'tblci']) {
      let c = t.get(o);
      c && (n[o] = c);
    }
    let e = t.get('ob_click_id') || t.get('obclid');
    e && (n.ob_click_id = e);
  }
  async function d(n, t) {
    let e = [],
      o = globalThis.tagEvSnapshot;
    if (typeof o == 'function') {
      let i = o();
      i && i.events && i.events.length && (e = e.concat(i.events));
    }
    let c = globalThis.tagInSnapshot;
    if (typeof c == 'function') {
      let i = c();
      i && i.events && i.events.length && (e = e.concat(i.events));
    }
    e.length && (n.ev = { events: e });
    let a = globalThis.tagCtxArm;
    typeof a == 'function' && t && a(t);
    let s = globalThis.tagCtxReady;
    typeof s == 'function' && (await s());
    let f = globalThis.tagCtxSnapshot;
    if (typeof f == 'function') {
      let i = f();
      i && (n.ctx = i);
    }
  }
  async function r(n) {
    let t = Number(n);
    if (!t || t <= 0) return;
    let e = performance.now();
    for (; performance.now() - e < t; ) await new Promise((o) => setTimeout(o, Math.min(t, 32)));
  }
  async function h(n) {
    let t = { campaign_id: n.campaignId, type: n.type },
      e =
        n.eventId ||
        (typeof crypto != 'undefined' && typeof crypto.randomUUID == 'function'
          ? crypto.randomUUID()
          : '');
    e && (t.event_id = e),
      n.clickId && (t.click_id = n.clickId),
      n.userId && (t.user_id = n.userId);
    let o = n.subs || {};
    for (let c = 1; c <= 30; c += 1) {
      let a = `sub${c}`;
      o[a] && (t[a] = o[a]);
    }
    return (
      l(t),
      await d(t, n.campaignId),
      await r(n.minDwellMs),
      fetch(n.endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(t),
        keepalive: !0,
        credentials: 'omit',
      })
    );
  }
  globalThis.sendEvent = h;
})();
