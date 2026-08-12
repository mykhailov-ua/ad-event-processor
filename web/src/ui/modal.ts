import { el, appendChildren } from '../lib/dom.js';

export type ModalChild = Node | string | null | false | undefined;

export type ModalOpts = {
  title: string;
  description?: string;
  onClose: () => void;
  body?: ModalChild[];
  actions?: ModalChild[];
  titleId?: string;
  footerClass?: string;
};

export type ModalHandle = {
  destroy: () => void;
};

/**
 * Mount a modal dialog overlay with focus trap and escape handling.
 */
export function mountModal(opts: ModalOpts): ModalHandle {
  const titleId = opts.titleId ?? `modal-title-${Math.random().toString(36).slice(2, 9)}`;

  const overlay = el('div', {
    className: 'modal-overlay',
    role: 'presentation',
    onClick: () => opts.onClose(),
  });

  const dialog = el('div', {
    className: 'modal',
    role: 'dialog',
    'aria-modal': 'true',
    'aria-labelledby': titleId,
    onClick: (e: Event) => e.stopPropagation(),
  });

  const header = el('div', { className: 'modal__header' },
    el('h2', { id: titleId, className: 'modal__title' }, opts.title),
  );

  const body = el('div', { className: 'modal__body' });
  if (opts.description) {
    body.appendChild(el('p', { className: 'modal__desc' }, opts.description));
  }
  if (opts.body) appendChildren(body, opts.body);

  const footer = el('div', { className: `modal__footer ${opts.footerClass ?? ''}`.trim() });
  if (opts.actions) appendChildren(footer, opts.actions);

  dialog.appendChild(header);
  if (body.childNodes.length > 0) dialog.appendChild(body);
  if (footer.childNodes.length > 0) dialog.appendChild(footer);
  overlay.appendChild(dialog);

  const onKey = (e: KeyboardEvent): void => {
    if (e.key === 'Escape') {
      opts.onClose();
      return;
    }
    if (e.key !== 'Tab') return;
    const focusable = dialog.querySelectorAll(
      'button:not([disabled]), input:not([disabled]), [tabindex]:not([tabindex="-1"])',
    );
    if (focusable.length === 0) return;
    const first = focusable[0] as HTMLElement;
    const last = focusable[focusable.length - 1] as HTMLElement;
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  };

  document.addEventListener('keydown', onKey);
  document.body.appendChild(overlay);

  const focusable = dialog.querySelector('button, input');
  if (focusable instanceof HTMLElement) focusable.focus();

  return {
    destroy() {
      document.removeEventListener('keydown', onKey);
      overlay.remove();
    },
  };
}
