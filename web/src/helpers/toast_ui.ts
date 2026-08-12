export type ToastMessage = {
  title: string;
  message: string;
  code?: string;
};

type ToastHandler = (msg: ToastMessage) => void;

let pushToast: ToastHandler | null = null;

/**
 * Register the global toast push handler used by pushToastMessage.
 */
export function setToastHandler(fn: ToastHandler | null): void {
  pushToast = fn;
}

/**
 * Enqueue a toast message when a handler is installed.
 */
export function pushToastMessage(msg: ToastMessage): void {
  if (pushToast) pushToast(msg);
}
