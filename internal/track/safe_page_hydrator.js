'use strict';
(() => {
  const verifyEndpoint = '/track/verify';
  const minEvents = 15;
  const minDwellMs = 1500;
  const maxEvents = 64;

  const events = [];
  let armed = false;
  let armTs = 0;
  let canvasHash64 = '';
  let canvasHashB64 = '';
  let audioHash64 = '';
  let webglVendor = '';
  let webglRenderer = '';
  let webrtcLocalIP = '';

  const honey = document.createElement('span');
  honey.textContent = '127.99';
  honey.setAttribute('data-honey-price', '1');
  honey.style.cssText = 'position:absolute;clip:rect(0,0,0,0);opacity:0;pointer-events:none';
  document.body.appendChild(honey);

  function parseCampaignId() {
    const params = new URLSearchParams(window.location.search);
    const fromQuery = params.get('campaign_id');
    if (fromQuery) {
      return fromQuery;
    }
    const meta = document.querySelector('meta[name="aed-campaign-id"]');
    return meta ? meta.getAttribute('content') : '';
  }

  function pushEvent(evt) {
    if (events.length >= maxEvents) {
      return;
    }
    events.push(evt);
  }

  function onPointer(e) {
    pushEvent({
      t: e.type === 'click' ? 'click' : 'mousemove',
      ts: Math.round(performance.now()),
      x: e.clientX | 0,
      y: e.clientY | 0,
    });
  }

  function onScroll() {
    pushEvent({
      t: 'scroll',
      ts: Math.round(performance.now()),
      x: window.scrollX | 0,
      y: window.scrollY | 0,
    });
  }

  function onTouch(e) {
    const touch = e.touches && e.touches[0];
    if (!touch) {
      return;
    }
    pushEvent({
      t: e.type,
      ts: Math.round(performance.now()),
      x: touch.clientX | 0,
      y: touch.clientY | 0,
      force: touch.force || 0,
      radius_x: touch.radiusX || 0,
      radius_y: touch.radiusY || 0,
    });
  }

  function armListeners() {
    if (armed) {
      return;
    }
    armed = true;
    armTs = performance.now();
    window.addEventListener('pointermove', onPointer, { passive: true });
    window.addEventListener('click', onPointer, { passive: true });
    window.addEventListener('scroll', onScroll, { passive: true });
    window.addEventListener('touchstart', onTouch, { passive: true });
    window.addEventListener('touchmove', onTouch, { passive: true });
    if (typeof globalThis.tagEvArm === 'function') {
      globalThis.tagEvArm();
    }
    if (typeof globalThis.tagInArm === 'function') {
      globalThis.tagInArm();
    }
  }

  async function sha256Hex(bytes) {
    if (!globalThis.crypto || !crypto.subtle) {
      return '';
    }
    const buf = await crypto.subtle.digest('SHA-256', bytes);
    const arr = new Uint8Array(buf);
    let out = '';
    for (let i = 0; i < arr.length; i += 1) {
      out += arr[i].toString(16).padStart(2, '0');
    }
    return out;
  }

  async function probeCanvasHashes() {
    try {
      const canvas = document.createElement('canvas');
      canvas.width = 64;
      canvas.height = 16;
      const ctx = canvas.getContext('2d');
      if (!ctx) {
        return;
      }
      ctx.fillStyle = '#000';
      ctx.fillText('aed-tc', 2, 12);
      const img = ctx.getImageData(0, 0, 64, 16);
      canvasHash64 = await sha256Hex(img.data);
      ctx.fillStyle = '#111';
      ctx.fillRect(0, 0, 4, 4);
      const imgB = ctx.getImageData(0, 0, 64, 16);
      canvasHashB64 = await sha256Hex(imgB.data);
    } catch (_err) {
      canvasHash64 = '';
      canvasHashB64 = '';
    }
  }

  async function probeAudioHash() {
    try {
      const Ctx = window.OfflineAudioContext || window.webkitOfflineAudioContext;
      if (!Ctx) {
        return;
      }
      const ctx = new Ctx(1, 5000, 44100);
      const osc = ctx.createOscillator();
      osc.type = 'triangle';
      osc.frequency.value = 10000;
      osc.connect(ctx.destination);
      osc.start(0);
      const rendered = await ctx.startRendering();
      const ch = rendered.getChannelData(0);
      const slice = ch.subarray(4500, 5000);
      const bytes = new Float32Array(slice.length);
      bytes.set(slice);
      audioHash64 = await sha256Hex(bytes.buffer);
    } catch (_err) {
      audioHash64 = '';
    }
  }

  function probeWebGL() {
    try {
      const canvas = document.createElement('canvas');
      const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
      if (!gl) {
        return;
      }
      const dbg = gl.getExtension('WEBGL_debug_renderer_info');
      if (dbg) {
        webglVendor = String(gl.getParameter(dbg.UNMASKED_VENDOR_WEBGL) || '');
        webglRenderer = String(gl.getParameter(dbg.UNMASKED_RENDERER_WEBGL) || '');
      }
    } catch (_err) {
      webglVendor = '';
      webglRenderer = '';
    }
  }

  function probeWebRTC() {
    return new Promise((resolve) => {
      try {
        const PC = window.RTCPeerConnection || window.webkitRTCPeerConnection;
        if (!PC) {
          resolve();
          return;
        }
        const pc = new PC({ iceServers: [] });
        pc.createDataChannel('aed');
        pc.onicecandidate = (ev) => {
          if (!ev || !ev.candidate || !ev.candidate.candidate) {
            return;
          }
          const m = ev.candidate.candidate.match(/(\d+\.\d+\.\d+\.\d+)/);
          if (m) {
            webrtcLocalIP = m[1];
          }
        };
        pc.createOffer()
          .then((offer) => pc.setLocalDescription(offer))
          .catch(() => {});
        setTimeout(() => {
          try {
            pc.close();
          } catch (_e) {
            /* ignore */
          }
          resolve();
        }, 400);
      } catch (_err) {
        resolve();
      }
    });
  }

  async function probeRuntimeDeep() {
    const runtime = {};
    try {
      const t0 = performance.now();
      const canvas = document.createElement('canvas');
      const gl = canvas.getContext('webgl');
      if (gl) {
        const vs = gl.createShader(gl.VERTEX_SHADER);
        gl.shaderSource(vs, 'attribute vec4 p;void main(){gl_Position=p;}');
        gl.compileShader(vs);
        runtime.shader_compile_ms = Math.min(
          65535,
          Math.max(0, Math.round(performance.now() - t0))
        );
      }
    } catch (_err) {}
    try {
      const noise = (0.1 + 0.2).toString();
      runtime.float_noise_hash = await sha256Hex(new TextEncoder().encode(noise));
    } catch (_err) {}
    try {
      const g0 = performance.now();
      void navigator.userAgent;
      runtime.navigator_getter_us = Math.min(
        65535,
        Math.max(1, Math.round((performance.now() - g0) * 1000))
      );
    } catch (_err) {
      runtime.navigator_getter_us = 0;
    }
    return runtime;
  }

  function runtimeDeepProbesEnabled() {
    const meta = document.querySelector('meta[name="aed-runtime-probes"]');
    return meta && meta.getAttribute('content') === '1';
  }

  function mergeTelemetryEvents() {
    const out = events.slice();
    const snapFn = globalThis.tagEvSnapshot;
    if (typeof snapFn === 'function') {
      const ext = snapFn();
      if (ext && ext.events && ext.events.length) {
        for (let i = 0; i < ext.events.length && out.length < maxEvents; i += 1) {
          const e = ext.events[i];
          out.push({
            t: e.t,
            ts: e.ts,
            x: e.x,
            y: e.y,
          });
        }
      }
    }
    const bioFn = globalThis.tagInSnapshot;
    if (typeof bioFn === 'function') {
      const bio = bioFn();
      if (bio && bio.events && bio.events.length) {
        for (let i = 0; i < bio.events.length && out.length < maxEvents; i += 1) {
          const e = bio.events[i];
          out.push({
            t: e.t,
            ts: e.ts,
            x: e.x | 0,
            y: e.y | 0,
          });
        }
      }
    }
    return out;
  }

  function buildFingerprint() {
    const nav = navigator;
    const perm =
      typeof Notification !== 'undefined' && Notification.permission
        ? Notification.permission
        : 'default';
    const mobile =
      /Mobi|Android|iPhone|iPad/i.test(nav.userAgent || '') ||
      (nav.maxTouchPoints && nav.maxTouchPoints > 1);
    return {
      ua: nav.userAgent || '',
      lang: nav.language || '',
      languages: nav.languages ? Array.prototype.slice.call(nav.languages) : [],
      platform: nav.platform || '',
      cores: nav.hardwareConcurrency || 0,
      screen: [screen.width | 0, screen.height | 0],
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || '',
      webdriver: !!nav.webdriver,
      webrtc_local_ip: webrtcLocalIP,
      webgl_vendor: webglVendor,
      webgl_renderer: webglRenderer,
      mobile,
      outer_width: window.outerWidth | 0,
      outer_height: window.outerHeight | 0,
      inner_width: window.innerWidth | 0,
      inner_height: window.innerHeight | 0,
      plugins_length: nav.plugins ? nav.plugins.length : 0,
      canvas_hash: canvasHash64,
      canvas_hash_a: canvasHash64,
      canvas_hash_b: canvasHashB64,
      audio_hash: audioHash64,
      notification_permission: perm,
      notification_query: perm,
    };
  }

  function graftVerifiedHtml(html) {
    if (!html) {
      return false;
    }
    const mount = document.getElementById('aed-mount');
    if (mount) {
      mount.innerHTML = html;
      document.documentElement.style.visibility = 'visible';
      return true;
    }
    document.open();
    document.write(html);
    document.close();
    return true;
  }

  async function requestServerUnlock(campaignId, snap) {
    await probeCanvasHashes();
    if (!audioHash64) {
      await probeAudioHash();
    }
    if (!webglVendor) {
      probeWebGL();
    }
    await probeWebRTC();

    const body = {
      campaign_id: campaignId,
      events: mergeTelemetryEvents(),
      fp: buildFingerprint(),
    };
    if (runtimeDeepProbesEnabled()) {
      body.fp.runtime_probes = await probeRuntimeDeep();
    }
    if (snap) {
      body.ctx = snap;
    }

    const resp = await fetch(verifyEndpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify(body),
    });
    if (!resp.ok) {
      return { ok: false };
    }
    const payload = await resp.json();
    if (!payload || !payload.success || !payload.html_content) {
      return { ok: false, code: payload && payload.code };
    }
    graftVerifiedHtml(payload.html_content);
    return { ok: true };
  }

  function readyForVerify(snap) {
    const dwellOk = performance.now() - armTs >= minDwellMs;
    const motionOk = events.length >= minEvents;
    const whenReady = globalThis.tagCtxReady;
    const cryptoOk = !whenReady || (snap && snap.ctx_mac && snap.pow_nonce);
    return dwellOk && motionOk && cryptoOk;
  }

  async function gateVerify() {
    const campaignId = parseCampaignId();
    armListeners();
    if (typeof globalThis.tagCtxArm === 'function') {
      globalThis.tagCtxArm(campaignId);
    }
    const whenReady = globalThis.tagCtxReady;
    if (typeof whenReady === 'function') {
      await whenReady();
    }

    for (let attempt = 0; attempt < 120; attempt += 1) {
      const snapFn = globalThis.tagCtxSnapshot;
      const snap = typeof snapFn === 'function' ? snapFn() : null;
      if (readyForVerify(snap)) {
        const result = await requestServerUnlock(campaignId, snap);
        if (result.ok) {
          return;
        }
        break;
      }
      await new Promise((r) => setTimeout(r, 250));
    }
    document.documentElement.style.visibility = 'visible';
  }

  document.documentElement.style.visibility = 'hidden';
  gateVerify().catch(() => {
    document.documentElement.style.visibility = 'visible';
  });
})();
