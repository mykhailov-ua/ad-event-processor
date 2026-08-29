'use strict';
(() => {
  const maxEvents = 64;
  const events = [];
  let armed = false;

  function push(evt) {
    if (events.length >= maxEvents) {
      return;
    }
    events.push(evt);
  }

  function onMouse(e) {
    push({ t: 'mousemove', ts: Date.now(), x: e.clientX | 0, y: e.clientY | 0 });
  }

  function onTouch(e) {
    const touch = e.touches && e.touches[0];
    if (!touch) {
      return;
    }
    push({ t: 'touchstart', ts: Date.now(), x: touch.clientX | 0, y: touch.clientY | 0 });
  }

  function onScroll() {
    push({ t: 'scroll', ts: Date.now(), x: window.scrollX | 0, y: window.scrollY | 0 });
  }

  function arm() {
    if (armed) {
      return;
    }
    armed = true;
    window.addEventListener('mousemove', onMouse, { passive: true });
    window.addEventListener('touchstart', onTouch, { passive: true });
    window.addEventListener('scroll', onScroll, { passive: true });
  }

  function snapshot() {
    return { events: events.slice() };
  }

  globalThis.trackTelemetryArm = arm;
  globalThis.trackTelemetrySnapshot = snapshot;
})();
