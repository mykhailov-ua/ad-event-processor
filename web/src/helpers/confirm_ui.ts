import type { ConfirmEntry } from './confirm_registry.js';

export type ConfirmRequest = {
  entry: ConfirmEntry;
  method: string;
  path: string;
  title?: string;
  description?: string;
};

type ConfirmHandler = (req: ConfirmRequest) => Promise<boolean>;

let handler: ConfirmHandler | null = null;

let confirmQueue: Promise<void> = Promise.resolve();

/**
 * Register the global confirm-dialog handler.
 */
export function setConfirmHandler(fn: ConfirmHandler): void {
  handler = fn;
}

/**
 * Run the confirm flow for a mutation request.
 */
export function requestConfirm(req: ConfirmRequest): Promise<boolean> {
  if (!handler) return Promise.resolve(true);
  const run = confirmQueue.then(() => handler!(req));
  confirmQueue = run.then(() => {}, () => {});
  return run;
}

/** Thrown when the operator cancels a confirm dialog. */
export class ConfirmCancelledError extends Error {
  constructor() {
    super('confirm cancelled');
    this.name = 'ConfirmCancelledError';
  }
}
