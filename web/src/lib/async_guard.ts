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

export function guardTelemetryReport(): GuardTelemetry {
  return { ...guardTelemetry };
}

export function guardTelemetryReset(): void {
  guardTelemetry.stale_write_prevented = 0;
  guardTelemetry.in_flight_rejected = 0;
}

export function shouldCommitAsyncResult(
  opGen: number,
  currentGen: number,
  destroyed = false
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
