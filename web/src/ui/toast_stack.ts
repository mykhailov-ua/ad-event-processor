import { el } from '../lib/dom.js';
import { setToastHandler, type ToastMessage } from '../helpers/toast_ui.js';
import { renderIcon } from './icon.js';

const MAX_TOASTS = 3;
const DISMISS_MS = 5000;

export type ToastStackHandle = {
  destroy: () => void;
};

/**
 * Mount a toast stack and wire it to the global toast handler.
 */
export function installToastStack(root: HTMLElement): ToastStackHandle {
  const stack = el('div', { className: 'toast-stack', role: 'status' });
  root.appendChild(stack);

  const items: Array<{ id: string; node: HTMLElement }> = [];

  function push(msg: ToastMessage): void {
    const id = crypto.randomUUID();
    const node = el('div', { className: 'toast' },
      renderIcon('info', { size: 16, className: 'toast__icon' }),
      el('div', { className: 'toast__content' },
        el('div', { className: 'toast__title' }, msg.title),
        el('div', { className: 'toast__message' }, msg.message),
      ),
    );
    items.push({ id, node });
    while (items.length > MAX_TOASTS) {
      const old = items.shift();
      old?.node.remove();
    }
    stack.appendChild(node);
    setTimeout(() => {
      const idx = items.findIndex((t) => t.id === id);
      if (idx >= 0) {
        items[idx].node.remove();
        items.splice(idx, 1);
      }
    }, DISMISS_MS);
  }

  setToastHandler(push);

  return {
    destroy() {
      setToastHandler(null);
      stack.remove();
    },
  };
}
