'use strict';
(() => {
  const maxSamples = 96;
  const maxRTT = 8;
  const navStart = performance.now();
  let armed = false;
  let dwellStart = 0;
  let footerReachMs = 0;
  let trustedTotal = 0;
  let trustedCount = 0;

  // Pointer ring (SoA): zero per-event object alloc; O(1) overwrite via head index.
  const ptrTs = new Float64Array(maxSamples);
  const ptrX = new Int32Array(maxSamples);
  const ptrY = new Int32Array(maxSamples);
  const ptrVx = new Float32Array(maxSamples);
  const ptrVy = new Float32Array(maxSamples);
  const ptrAccel = new Float32Array(maxSamples);
  const ptrTrusted = new Uint8Array(maxSamples);
  let ptrHead = 0;
  let ptrCount = 0;

  const scrTs = new Float64Array(maxSamples);
  const scrY = new Int32Array(maxSamples);
  const scrVel = new Float32Array(maxSamples);
  const scrJerk = new Float32Array(maxSamples);
  let scrHead = 0;
  let scrCount = 0;

  const touchDt = new Float32Array(maxSamples);
  let touchHead = 0;
  let touchCount = 0;
  let lastTouchTs = 0;

  const rttBuf = new Uint16Array(maxRTT);
  let rttHead = 0;
  let rttCount = 0;

  let canvasHash = '';
  let audioHash = '';
  let webglHash = '';

  function ringLastIdx(head, count, cap) {
    if (count <= 0) {
      return -1;
    }
    return (head - 1 + cap) % cap;
  }

  function ringWriteIdx(head, count, cap) {
    const idx = head;
    return {
      idx,
      head: (head + 1) % cap,
      count: count < cap ? count + 1 : cap,
    };
  }

  function fnv1aU32(h, v) {
    return (h ^ v) + ((h ^ v) * 0x01000193) >>> 0;
  }

  function fnv1aBytes(bytes, off, len) {
    let h = 0x811c9dc5;
    const end = off + len;
    for (let i = off; i < end; i += 1) {
      h = fnv1aU32(h, bytes[i]);
    }
    return ('00000000' + h.toString(16)).slice(-8);
  }

  function fnv1aAscii(str) {
    let h = 0x811c9dc5;
    for (let i = 0; i < str.length; i += 1) {
      h = fnv1aU32(h, str.charCodeAt(i) & 0xff);
    }
    return ('00000000' + h.toString(16)).slice(-8);
  }

  function coeffVarMilliFromRing(values, head, count, cap) {
    if (count < 3) {
      return 0;
    }
    const start = count < cap ? 0 : head;
    let sum = 0;
    for (let i = 0; i < count; i += 1) {
      const idx = count < cap ? i : (start + i) % cap;
      sum += values[idx];
    }
    const mean = sum / count;
    if (mean <= 0) {
      return 0;
    }
    let varSum = 0;
    for (let i = 0; i < count; i += 1) {
      const idx = count < cap ? i : (start + i) % cap;
      const d = values[idx] - mean;
      varSum += d * d;
    }
    const std = Math.sqrt(varSum / (count - 1));
    return Math.round((std / mean) * 1000);
  }

  function coeffVarMilliFromTsRing(tsRing, head, count, cap) {
    if (count < 4) {
      return 0;
    }
    const start = count < cap ? 0 : head;
    const tmp = new Float32Array(count - 1);
    let n = 0;
    let prev = 0;
    for (let i = 0; i < count; i += 1) {
      const idx = count < cap ? i : (start + i) % cap;
      const ts = tsRing[idx];
      if (i > 0 && ts > prev) {
        tmp[n] = ts - prev;
        n += 1;
      }
      prev = ts;
    }
    if (n < 3) {
      return 0;
    }
    let sum = 0;
    for (let i = 0; i < n; i += 1) {
      sum += tmp[i];
    }
    const mean = sum / n;
    if (mean <= 0) {
      return 0;
    }
    let varSum = 0;
    for (let i = 0; i < n; i += 1) {
      const d = tmp[i] - mean;
      varSum += d * d;
    }
    const std = Math.sqrt(varSum / (n - 1));
    return Math.round((std / mean) * 1000);
  }

  function onPointerMove(e) {
    const ts = performance.now();
    trustedTotal += 1;
    trustedCount += e.isTrusted ? 1 : 0;

    const lastIdx = ringLastIdx(ptrHead, ptrCount, maxSamples);
    let vx = 0;
    let vy = 0;
    let accel = 0;
    if (lastIdx >= 0) {
      const dt = ts - ptrTs[lastIdx];
      if (dt > 0) {
        vx = (e.clientX - ptrX[lastIdx]) / dt;
        vy = (e.clientY - ptrY[lastIdx]) / dt;
        const lvx = ptrVx[lastIdx];
        const lvy = ptrVy[lastIdx];
        if (lvx !== 0 || lvy !== 0) {
          const dvx = vx - lvx;
          const dvy = vy - lvy;
          accel = Math.sqrt(dvx * dvx + dvy * dvy) / dt;
        }
      }
    }

    const slot = ringWriteIdx(ptrHead, ptrCount, maxSamples);
    ptrTs[slot.idx] = ts;
    ptrX[slot.idx] = e.clientX | 0;
    ptrY[slot.idx] = e.clientY | 0;
    ptrVx[slot.idx] = vx;
    ptrVy[slot.idx] = vy;
    ptrAccel[slot.idx] = accel;
    ptrTrusted[slot.idx] = e.isTrusted ? 1 : 0;
    ptrHead = slot.head;
    ptrCount = slot.count;
  }

  function onTouchStart(e) {
    const ts = performance.now();
    trustedTotal += 1;
    trustedCount += e.isTrusted ? 1 : 0;
    if (lastTouchTs > 0) {
      const slot = ringWriteIdx(touchHead, touchCount, maxSamples);
      touchDt[slot.idx] = ts - lastTouchTs;
      touchHead = slot.head;
      touchCount = slot.count;
    }
    lastTouchTs = ts;
    const touch = e.touches && e.touches[0];
    if (!touch) {
      return;
    }
    const slot = ringWriteIdx(ptrHead, ptrCount, maxSamples);
    ptrTs[slot.idx] = ts;
    ptrX[slot.idx] = touch.clientX | 0;
    ptrY[slot.idx] = touch.clientY | 0;
    ptrVx[slot.idx] = 0;
    ptrVy[slot.idx] = 0;
    ptrAccel[slot.idx] = 0;
    ptrTrusted[slot.idx] = e.isTrusted ? 1 : 0;
    ptrHead = slot.head;
    ptrCount = slot.count;
  }

  function onScroll() {
    const ts = performance.now();
    const y = window.scrollY | 0;
    const lastIdx = ringLastIdx(scrHead, scrCount, maxSamples);
    let vel = 0;
    let jerk = 0;
    if (lastIdx >= 0) {
      const dt = ts - scrTs[lastIdx];
      if (dt > 0) {
        vel = (y - scrY[lastIdx]) / dt;
        const lastVel = scrVel[lastIdx];
        if (lastVel !== 0) {
          jerk = Math.abs((vel - lastVel) / dt);
        }
      }
    }
    const slot = ringWriteIdx(scrHead, scrCount, maxSamples);
    scrTs[slot.idx] = ts;
    scrY[slot.idx] = y;
    scrVel[slot.idx] = vel;
    scrJerk[slot.idx] = jerk;
    scrHead = slot.head;
    scrCount = slot.count;

    if (!footerReachMs) {
      const doc = document.documentElement;
      const maxScroll = (doc ? doc.scrollHeight : 0) - window.innerHeight;
      if (maxScroll > 0 && y >= maxScroll * 0.92) {
        footerReachMs = Math.round(ts - navStart);
      }
    }
  }

  function canvasFingerprint() {
    try {
      const canvas = document.createElement('canvas');
      canvas.width = 240;
      canvas.height = 60;
      const ctx = canvas.getContext('2d', { willReadFrequently: true });
      if (!ctx) {
        return '';
      }
      ctx.textBaseline = 'alphabetic';
      ctx.fillStyle = '#f60';
      ctx.fillRect(0, 0, 120, 60);
      ctx.fillStyle = '#069';
      ctx.font = '14px Arial';
      ctx.fillText('aed-tc', 2, 15);
      ctx.strokeStyle = 'rgba(102,204,0,0.8)';
      ctx.arc(60, 30, 18, 0, Math.PI * 2);
      ctx.stroke();
      const pixels = ctx.getImageData(0, 0, 240, 60).data;
      return fnv1aBytes(pixels, 0, pixels.length);
    } catch (_err) {
      return '';
    }
  }

  function webglFingerprint() {
    try {
      const canvas = document.createElement('canvas');
      const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
      if (!gl) {
        return '';
      }
      const dbg = gl.getExtension('WEBGL_debug_renderer_info');
      const vendor = dbg ? gl.getParameter(dbg.UNMASKED_VENDOR_WEBGL) : gl.getParameter(gl.VENDOR);
      const renderer = dbg ? gl.getParameter(dbg.UNMASKED_RENDERER_WEBGL) : gl.getParameter(gl.RENDERER);
      const vs = gl.createShader(gl.VERTEX_SHADER);
      const fs = gl.createShader(gl.FRAGMENT_SHADER);
      gl.shaderSource(vs, 'attribute vec2 p;void main(){gl_Position=vec4(p,0.,1.);}');
      gl.shaderSource(fs, 'precision mediump float;void main(){gl_FragColor=vec4(0.13,0.47,0.71,1.);}');
      gl.compileShader(vs);
      gl.compileShader(fs);
      const prog = gl.createProgram();
      gl.attachShader(prog, vs);
      gl.attachShader(prog, f);
      gl.linkProgram(prog);
      gl.useProgram(prog);
      gl.drawArrays(gl.TRIANGLES, 0, 3);
      const pixels = new Uint8Array(16);
      gl.readPixels(0, 0, 2, 2, gl.RGBA, gl.UNSIGNED_BYTE, pixels);
      const meta = fnv1aAscii(String(vendor) + '|' + String(renderer));
      const px = fnv1aBytes(pixels, 0, pixels.length);
      return fnv1aAscii(meta + px);
    } catch (_err) {
      return '';
    }
  }

  function audioFingerprint(cb) {
    try {
      const Ctx = window.OfflineAudioContext || window.webkitOfflineAudioContext;
      if (!Ctx) {
        cb('');
        return;
      }
      const ctx = new Ctx(1, 44100, 44100);
      const osc = ctx.createOscillator();
      osc.type = 'triangle';
      osc.frequency.value = 10000;
      const comp = ctx.createDynamicsCompressor();
      comp.threshold.value = -50;
      comp.knee.value = 40;
      comp.ratio.value = 12;
      comp.attack.value = 0;
      comp.release.value = 0.25;
      osc.connect(comp);
      comp.connect(ctx.destination);
      osc.start(0);
      ctx.startRendering();
      ctx.oncomplete = (ev) => {
        const buf = ev.renderedBuffer.getChannelData(0);
        const view = new Uint8Array(buf.buffer, buf.byteOffset + 4500 * 4, 500 * 4);
        cb(fnv1aBytes(view, 0, view.length));
      };
    } catch (_err) {
      cb('');
    }
  }

  function probeRTT() {
    if (rttCount >= maxRTT) {
      return;
    }
    const start = performance.now();
    const img = new Image();
    img.onload = img.onerror = () => {
      if (rttCount >= maxRTT) {
        return;
      }
      const slot = ringWriteIdx(rttHead, rttCount, maxRTT);
      rttBuf[slot.idx] = Math.round(performance.now() - start);
      rttHead = slot.head;
      rttCount = slot.count;
    };
    img.src = '/favicon.ico?rtt=' + start;
  }

  let rafCvMilli = 0;
  let challengeToken = '';
  let powNonce = 0;
  let telemetryMac = '';
  let sealedSnapshot = null;
  let readyPromise = null;
  let campaignID = '';

  function b64urlDecode(str) {
    const pad = '='.repeat((4 - (str.length % 4)) % 4);
    const b64 = (str + pad).replace(/-/g, '+').replace(/_/g, '/');
    const bin = atob(b64);
    const out = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i += 1) {
      out[i] = bin.charCodeAt(i);
    }
    return out;
  }

  function decodeChallenge(token) {
    const raw = b64urlDecode(token);
    if (raw.length < 34) {
      return null;
    }
    return { salt: raw.slice(17, 33), difficulty: raw[33] };
  }

  async function solvePoW(salt, difficulty) {
    if (!crypto || !crypto.subtle) {
      return 0;
    }
    let nonce = 0;
    while (nonce < 2000000) {
      const buf = new Uint8Array(20);
      buf.set(salt, 0);
      buf[16] = (nonce >>> 24) & 255;
      buf[17] = (nonce >>> 16) & 255;
      buf[18] = (nonce >>> 8) & 255;
      buf[19] = nonce & 255;
      const hash = await crypto.subtle.digest('SHA-256', buf);
      const view = new Uint8Array(hash);
      let ok = true;
      for (let z = 0; z < difficulty; z += 1) {
        if (view[z] !== 0) {
          ok = false;
          break;
        }
      }
      if (ok) {
        return nonce;
      }
      nonce += 1;
    }
    return 0;
  }

  function hex16(bytes) {
    let out = '';
    for (let i = 0; i < 16 && i < bytes.length; i += 1) {
      out += ('0' + bytes[i].toString(16)).slice(-2);
    }
    return out;
  }

  async function deriveMacKey(challengeTok, nonce) {
    const enc = new TextEncoder();
    const tok = enc.encode(challengeTok);
    const data = new Uint8Array(tok.length + 4);
    data.set(tok, 0);
    data[tok.length] = (nonce >>> 24) & 255;
    data[tok.length + 1] = (nonce >>> 16) & 255;
    data[tok.length + 2] = (nonce >>> 8) & 255;
    data[tok.length + 3] = nonce & 255;
    const keyMat = await crypto.subtle.digest('SHA-256', data);
    return crypto.subtle.importKey('raw', keyMat, { name: 'HMAC', hash: 'SHA-256' }, false, ['sign']);
  }

  async function signTelemetry(challengeTok, nonce, dwellMs, pointerCV, rafCV, runtimeLeak, automationLeak) {
    const payload = new Uint8Array(10);
    payload[0] = (dwellMs >>> 24) & 255;
    payload[1] = (dwellMs >>> 16) & 255;
    payload[2] = (dwellMs >>> 8) & 255;
    payload[3] = dwellMs & 255;
    payload[4] = (pointerCV >>> 8) & 255;
    payload[5] = pointerCV & 255;
    payload[6] = (rafCV >>> 8) & 255;
    payload[7] = rafCV & 255;
    payload[8] = runtimeLeak;
    payload[9] = automationLeak;
    const key = await deriveMacKey(challengeTok, nonce);
    const sig = await crypto.subtle.sign('HMAC', key, payload);
    return hex16(new Uint8Array(sig));
  }

  function isWebdriverGetterTampered() {
    const desc = Object.getOwnPropertyDescriptor(Navigator.prototype, 'webdriver');
    if (!desc || typeof desc.get !== 'function') {
      return false;
    }
    try {
      desc.get.call({});
      return true;
    } catch (e) {
      return !(e instanceof TypeError);
    }
  }

  function isNavigatorWebdriverShadowed() {
    const inst = Object.getOwnPropertyDescriptor(navigator, 'webdriver');
    return inst !== undefined && inst.configurable === false;
  }

  function probeGetterStackLeak() {
    const desc = Object.getOwnPropertyDescriptor(Navigator.prototype, 'languages');
    if (!desc || typeof desc.get !== 'function') {
      return false;
    }
    try {
      desc.get.call({});
    } catch (e) {
      if (e && e.stack && /puppeteer|playwright|evaluation|__puppeteer|cdp/i.test(e.stack)) {
        return true;
      }
    }
    return false;
  }

  function measureRafJitter() {
    return new Promise((resolve) => {
      const deltas = new Float32Array(12);
      let last = performance.now();
      let count = 0;
      function step(ts) {
        if (count > 0) {
          deltas[count - 1] = ts - last;
        }
        last = ts;
        count += 1;
        if (count <= 12) {
          requestAnimationFrame(step);
        } else {
          resolve(coeffVarMilliFromRing(deltas, 0, 11, 11));
        }
      }
      requestAnimationFrame(step);
    });
  }

  function nativeFnLooksTampered(fn) {
    try {
      const src = Function.prototype.toString.call(fn);
      return src.indexOf('[native code]') < 0;
    } catch (_err) {
      return true;
    }
  }

  function detectRuntimeTampering() {
    let leak = 0;
    if (isWebdriverGetterTampered()) {
      leak |= 1;
    }
    if (isNavigatorWebdriverShadowed()) {
      leak |= 2;
    }
    if (probeGetterStackLeak()) {
      leak |= 4;
    }
    if (rafCvMilli > 800) {
      leak |= 8;
    }
    return leak;
  }

  function detectAutomation() {
    let leak = 0;
    let webdriver = 0;
    try {
      const wdDesc = Object.getOwnPropertyDescriptor(Navigator.prototype, 'webdriver');
      if (navigator.webdriver) {
        webdriver = 1;
        leak |= 1 << 4;
      } else if (wdDesc && typeof wdDesc.get === 'function') {
        webdriver = 1;
        leak |= 1 << 4;
      }

      if (document && document.$cdc_asdjflasutopfhvcZLmcfl_) {
        leak |= 1 << 0;
      }
      if (window._phantom || window.callPhantom) {
        leak |= 1 << 1;
      }
      if (window.__playwright || window.__pwInitScripts) {
        leak |= 1 << 2;
      }
      if (window.__selenium_unwrapped || window.__webdriver_evaluate || window.__driver_evaluate) {
        leak |= 1 << 3;
      }

      if (
        nativeFnLooksTampered(Function.prototype.toString) ||
        nativeFnLooksTampered(navigator.permissions.query)
      ) {
        leak |= 1 << 5;
      }

      if (
        (window.outerWidth === 0 || window.outerHeight === 0) ||
        (window.innerWidth > 0 &&
          window.innerWidth === window.outerWidth &&
          window.innerHeight === window.outerHeight)
      ) {
        leak |= 1 << 6;
      }

      const stackErr = new Error('aed-probe');
      if (stackErr.stack && /puppeteer|playwright|selenium|webdriver|cdp/i.test(stackErr.stack)) {
        leak |= 1 << 7;
      }
    } catch (_err) {
      leak = 0;
    }
    return { webdriver, automation_leak: leak };
  }

  function snapshotPointerSpeedCV() {
    if (ptrCount < 3) {
      return 0;
    }
    const scratch = new Float32Array(ptrCount);
    const start = ptrCount < maxSamples ? 0 : ptrHead;
    for (let i = 0; i < ptrCount; i += 1) {
      const idx = ptrCount < maxSamples ? i : (start + i) % maxSamples;
      const vx = ptrVx[idx];
      const vy = ptrVy[idx];
      scratch[i] = Math.sqrt(vx * vx + vy * vy);
    }
    return coeffVarMilliFromRing(scratch, 0, ptrCount, ptrCount);
  }

  function snapshotScrollVelCV() {
    const scratch = new Float32Array(scrCount);
    const start = scrCount < maxSamples ? 0 : scrHead;
    for (let i = 0; i < scrCount; i += 1) {
      const idx = scrCount < maxSamples ? i : (start + i) % maxSamples;
      scratch[i] = Math.abs(scrVel[idx]);
    }
    return coeffVarMilliFromRing(scratch, 0, scrCount, scrCount);
  }

  function snapshotScrollJerkCV() {
    const scratch = new Float32Array(scrCount);
    const start = scrCount < maxSamples ? 0 : scrHead;
    for (let i = 0; i < scrCount; i += 1) {
      const idx = scrCount < maxSamples ? i : (start + i) % maxSamples;
      scratch[i] = scrJerk[idx];
    }
    return coeffVarMilliFromRing(scratch, 0, scrCount, scrCount);
  }

  function snapshotRTTSamples() {
    const out = new Array(rttCount);
    const start = rttCount < maxRTT ? 0 : rttHead;
    for (let i = 0; i < rttCount; i += 1) {
      const idx = rttCount < maxRTT ? i : (start + i) % maxRTT;
      out[i] = rttBuf[idx];
    }
    return out;
  }

  function buildSnapshotBody() {
    const dwellMs = Math.round(performance.now() - (dwellStart || navStart));
    const trustedRatioMilli =
      trustedTotal > 0 ? Math.round((trustedCount / trustedTotal) * 1000) : 1000;
    const auto = detectAutomation();
    const runtimeLeak = detectRuntimeTampering();
    return {
      nav_pt_ms: Math.round(navStart),
      dwell_ms: dwellMs,
      footer_reach_ms: footerReachMs,
      trusted_ratio_milli: trustedRatioMilli,
      pointer_cv_milli: snapshotPointerSpeedCV(),
      pointer_dt_cv_milli: coeffVarMilliFromTsRing(ptrTs, ptrHead, ptrCount, maxSamples),
      scroll_cv_milli: snapshotScrollVelCV(),
      scroll_jerk_milli: snapshotScrollJerkCV(),
      touch_interval_cv_milli: coeffVarMilliFromRing(touchDt, touchHead, touchCount, maxSamples),
      raf_cv_milli: rafCvMilli,
      webdriver: auto.webdriver,
      automation_leak: auto.automation_leak,
      runtime_leak: runtimeLeak,
      canvas_hash: canvasHash,
      audio_hash: audioHash,
      webgl_hash: webglHash,
      rtt_samples: snapshotRTTSamples(),
      challenge_token: challengeToken,
      pow_nonce: powNonce,
      telemetry_mac: telemetryMac,
    };
  }

  async function fetchChallenge(id) {
    const res = await fetch('/track/antifraud/challenge?campaign_id=' + encodeURIComponent(id), {
      credentials: 'omit',
      cache: 'no-store',
    });
    if (!res.ok) {
      return null;
    }
    const data = await res.json();
    return data && data.challenge_token ? data : null;
  }

  async function bootstrapCrypto(id) {
    const ch = await fetchChallenge(id);
    if (!ch) {
      return;
    }
    challengeToken = ch.challenge_token;
    const decoded = decodeChallenge(challengeToken);
    if (!decoded) {
      return;
    }
    rafCvMilli = await measureRafJitter();
    powNonce = await solvePoW(decoded.salt, decoded.difficulty || 2);
    const body = buildSnapshotBody();
    telemetryMac = await signTelemetry(
      challengeToken,
      powNonce,
      body.dwell_ms,
      body.pointer_cv_milli,
      body.raf_cv_milli,
      body.runtime_leak,
      body.automation_leak
    );
    body.pow_nonce = powNonce;
    body.telemetry_mac = telemetryMac;
    sealedSnapshot = Object.freeze(body);
  }

  function arm(id) {
    if (armed) {
      return;
    }
    armed = true;
    campaignID = id || campaignID;
    dwellStart = performance.now();
    window.addEventListener('pointermove', onPointerMove, { passive: true });
    window.addEventListener('touchstart', onTouchStart, { passive: true });
    window.addEventListener('scroll', onScroll, { passive: true });
    canvasHash = canvasFingerprint();
    webglHash = webglFingerprint();
    audioFingerprint((h) => {
      audioHash = h;
    });
    for (let i = 0; i < 4; i += 1) {
      setTimeout(probeRTT, 400 + i * 600);
    }
    if (campaignID) {
      readyPromise = bootstrapCrypto(campaignID);
    } else {
      readyPromise = Promise.resolve();
    }
  }

  function snapshot() {
    if (sealedSnapshot) {
      return sealedSnapshot;
    }
    return buildSnapshotBody();
  }

  function whenReady() {
    if (!readyPromise) {
      return Promise.resolve();
    }
    return readyPromise;
  }

  Object.defineProperty(globalThis, 'trackAntifraudArm', { value: arm, writable: false, configurable: false });
  Object.defineProperty(globalThis, 'trackAntifraudSnapshot', { value: snapshot, writable: false, configurable: false });
  Object.defineProperty(globalThis, 'trackAntifraudWhenReady', { value: whenReady, writable: false, configurable: false });
})();
