export type ToastMessage = {
  title: string;
  message: string;
  code?: string;
};

type ToastHandler = (msg: ToastMessage) => void;

let pushToast: ToastHandler | null = null;

export function setToastHandler(fn: ToastHandler | null): void {
  pushToast = fn;
}

export function pushToastMessage(msg: ToastMessage): void {
  if (pushToast) pushToast(msg);
}
