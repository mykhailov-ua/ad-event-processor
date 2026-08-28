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

export function setConfirmHandler(fn: ConfirmHandler): void {
  handler = fn;
}

export function requestConfirm(req: ConfirmRequest): Promise<boolean> {
  if (!handler) return Promise.resolve(true);
  const run = confirmQueue.then(() => handler!(req));
  confirmQueue = run.then(
    () => {},
    () => {}
  );
  return run;
}

export class ConfirmCancelledError extends Error {
  constructor() {
    super('confirm cancelled');
    this.name = 'ConfirmCancelledError';
  }
}
