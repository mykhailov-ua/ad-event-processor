/**
 * Monotonic generation counter for async commit guards.
 * Invariant: only the latest generation may commit (opGen === currentGen).
 */
export type GenerationGuard = {
  next: () => number;
  isCurrent: (id: number) => boolean;
  invalidate: () => void;
  current: () => number;
};

export function createGenerationGuard(): GenerationGuard {
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

export type GuardTelemetry = {
  stale_write_prevented: number;
  in_flight_rejected: number;
};

const guardTelemetry: GuardTelemetry = {
  stale_write_prevented: 0,
  in_flight_rejected: 0,
};

/**
 * Return async guard rejection counters.
 */
export function guardTelemetryReport(): GuardTelemetry {
  return { ...guardTelemetry };
}

/**
 * Reset async guard telemetry counters.
 */
export function guardTelemetryReset(): void {
  guardTelemetry.stale_write_prevented = 0;
  guardTelemetry.in_flight_rejected = 0;
}

/**
 * Return whether an async side-effect may commit to view state.
 */
export function shouldCommitAsyncResult(
  opGen: number,
  currentGen: number,
  destroyed = false,
): boolean {
  const ok = !destroyed && opGen === currentGen;
  if (!ok && !destroyed && opGen !== currentGen) {
    guardTelemetry.stale_write_prevented += 1;
  }
  return ok;
}

export type InFlightGuard = {
  tryAcquire: () => boolean;
  release: () => void;
  busy: () => boolean;
};

/**
 * Single-flight guard: at most one in-flight mutation at a time.
 */
export function createInFlightGuard(): InFlightGuard {
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
