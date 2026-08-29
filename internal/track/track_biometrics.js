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

  function onTouch(e) {
    const touch = e.touches && e.touches[0];
    if (!touch) {
      return;
    }
    push({ t: e.type, ts: Date.now(), x: touch.clientX | 0, y: touch.clientY | 0 });
  }

  function onOrientation(e) {
    push({ t: 'deviceorientation', ts: Date.now(), x: e.beta | 0, y: e.gamma | 0 });
  }

  function arm() {
    if (armed) {
      return;
    }
    armed = true;
    window.addEventListener('touchstart', onTouch, { passive: true });
    window.addEventListener('touchmove', onTouch, { passive: true });
    window.addEventListener('deviceorientation', onOrientation, { passive: true });
  }

  function snapshot() {
    return { events: events.slice() };
  }

  globalThis.trackBiometricsArm = arm;
  globalThis.trackBiometricsSnapshot = snapshot;
})();
