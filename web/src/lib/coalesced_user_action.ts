export const DEFAULT_COALESCE_WINDOW_MS = 500;

export type CoalesceUserActionInput = {
  lastFiredAtMs: number;
  nowMs: number;
  windowMs: number;
  inFlight: boolean;
  inFlightGuard: boolean;
};

export type CoalesceUserActionResult = 'allow' | 'skip_in_flight' | 'skip_window';

export function coalesceUserAction(input: CoalesceUserActionInput): CoalesceUserActionResult {
  if (input.inFlightGuard && input.inFlight) {
    return 'skip_in_flight';
  }
  if (input.nowMs - input.lastFiredAtMs < input.windowMs) {
    return 'skip_window';
  }
  return 'allow';
}

export type CoalescedCallbackOptions = {
  windowMs?: number;
  inFlightGuard?: boolean;
  inFlight?: boolean;
};
