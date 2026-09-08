'use strict';
(() => {
  const maxEvents = 64;
  const events = [];
  let armed = false;
  const navStart = performance.now();

  function monoTs() {
    return Math.round((performance.now() - navStart) * 1000);
  }

  function push(evt) {
    if (events.length >= maxEvents) {
      return;
    }
    events.push(evt);
  }

  function pushPointer(type, e) {
    const fx = e.clientX;
    const fy = e.clientY;
    push({
      t: type,
      ts: monoTs(),
      x: fx | 0,
      y: fy | 0,
      fx: fx,
      fy: fy,
      trusted: e.isTrusted ? 1 : 0,
    });
  }

  function onPointerDown(e) {
    pushPointer('pointerdown', e);
  }

  function onClick(e) {
    pushPointer('click', e);
  }

  function onMouse(e) {
    const fx = e.clientX;
    const fy = e.clientY;
    push({
      t: 'mousemove',
      ts: monoTs(),
      x: fx | 0,
      y: fy | 0,
      fx: fx,
      fy: fy,
      trusted: e.isTrusted ? 1 : 0,
    });
  }

  function onTouch(e) {
    const touch = e.touches && e.touches[0];
    if (!touch) {
      return;
    }
    const fx = touch.clientX;
    const fy = touch.clientY;
    push({
      t: 'touchstart',
      ts: monoTs(),
      x: fx | 0,
      y: fy | 0,
      fx: fx,
      fy: fy,
      trusted: e.isTrusted ? 1 : 0,
    });
  }

  function onScroll() {
    const fx = window.scrollX;
    const fy = window.scrollY;
    push({
      t: 'scroll',
      ts: monoTs(),
      x: fx | 0,
      y: fy | 0,
      fx: fx,
      fy: fy,
      trusted: 1,
    });
  }

  function onKeydown(e) {
    push({
      t: 'keydown',
      ts: monoTs(),
      x: 0,
      y: 0,
      trusted: e.isTrusted ? 1 : 0,
    });
  }

  function onVisibility() {
    push({
      t: 'visibilitychange',
      ts: monoTs(),
      x: document.visibilityState === 'visible' ? 1 : 0,
      y: 0,
      trusted: 1,
    });
  }

  function arm() {
    if (armed) {
      return;
    }
    armed = true;
    window.addEventListener('pointerdown', onPointerDown, { passive: true });
    window.addEventListener('click', onClick, { passive: true });
    window.addEventListener('mousemove', onMouse, { passive: true });
    window.addEventListener('touchstart', onTouch, { passive: true });
    window.addEventListener('scroll', onScroll, { passive: true });
    window.addEventListener('keydown', onKeydown, { passive: true });
    document.addEventListener('visibilitychange', onVisibility, { passive: true });
  }

  function snapshot() {
    return { events: events.slice() };
  }

  globalThis.trackTelemetryArm = arm;
  globalThis.trackTelemetrySnapshot = snapshot;
})();
