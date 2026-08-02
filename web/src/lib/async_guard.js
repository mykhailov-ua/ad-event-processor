/**
 * Monotonic generation counter for async commit guards.
 * Invariant: only the latest generation may commit (opGen === currentGen).
 *
 * @returns {{ next: () => number, isCurrent: (id: number) => boolean, invalidate: () => void, current: () => number }}
 */
export function createGenerationGuard() {
  let gen = 0;
  return {
    next() {
      return ++gen;
    },
    isCurrent(id) {
      return id === gen;
    },
    invalidate() {
      gen += 1;
    },
    current() {
      return gen;
    },
  };
}

/** @type {{ stale_write_prevented: number, in_flight_rejected: number }} */
const guardTelemetry = {
  stale_write_prevented: 0,
  in_flight_rejected: 0,
};

/**
 * Return async guard rejection counters.
 *
 * @returns {{ stale_write_prevented: number, in_flight_rejected: number }}
 */
export function guardTelemetryReport() {
  return { ...guardTelemetry };
}

/**
 * Reset async guard telemetry counters.
 *
 * @returns {void}
 */
export function guardTelemetryReset() {
  guardTelemetry.stale_write_prevented = 0;
  guardTelemetry.in_flight_rejected = 0;
}

/**
 * Return whether an async side-effect may commit to view state.
 *
 * @param {number} opGen generation captured at operation start
 * @param {number} currentGen latest generation after overlapping ops
 * @param {boolean} [destroyed]
 * @returns {boolean}
 */
export function shouldCommitAsyncResult(opGen, currentGen, destroyed = false) {
  const ok = !destroyed && opGen === currentGen;
  if (!ok && !destroyed && opGen !== currentGen) {
    guardTelemetry.stale_write_prevented += 1;
  }
  return ok;
}

/**
 * Single-flight guard: at most one in-flight mutation at a time.
 *
 * @returns {{ tryAcquire: () => boolean, release: () => void, busy: () => boolean }}
 */
export function createInFlightGuard() {
  let busy = false;
  return {
    tryAcquire() {
      if (busy) {
        guardTelemetry.in_flight_rejected += 1;
        return false;
      }
      busy = true;
      return true;
    },
    release() {
      busy = false;
    },
    busy() {
      return busy;
    },
  };
}
