import { el } from '../lib/dom.js';
import { mountModal, type ModalHandle } from './modal.js';
import { renderCheckbox } from './checkbox.js';

const STRONG_TOKEN = 'DELETE';

export type ConfirmDialogOpts = {
  level: string;
  title?: string;
  description?: string;
  layout?: 'vertical' | 'horizontal';
  onConfirm: () => void;
  onCancel: () => void;
};

/**
 * Mount a confirm dialog for the given registry level.
 */
export function mountConfirmDialog(opts: ConfirmDialogOpts): ModalHandle {
  const isStrong = opts.level === 'strong';
  const isDestructive = opts.level === 'destructive' || opts.level === 'financial';

  const defaultTitle =
    opts.level === 'retry'
      ? 'Retry operation?'
      : opts.level === 'financial'
        ? 'Confirm financial operation'
        : opts.level === 'destructive'
          ? 'Confirm action'
          : opts.level === 'strong'
            ? 'Critical action'
            : 'Confirm action';

  let typed = '';
  let strongChecked = false;

  let confirmBtn: HTMLButtonElement | null = null;

  function updateConfirmBtn(): void {
    if (!confirmBtn) return;
    const canConfirm = isStrong
      ? typed === STRONG_TOKEN && strongChecked
      : true;
    confirmBtn.disabled = !canConfirm;
  }

  const body: HTMLElement[] = [];

  if (isStrong) {
    const field = el('div', { className: 'form-field' });

    field.appendChild(el('label', {
      className: 'form-label',
      htmlFor: 'confirm-strong-token',
    }, `Type ${STRONG_TOKEN} to confirm`));

    const tokenInput = el('input', {
      id: 'confirm-strong-token',
      type: 'text',
      className: 'form-input',
      autocomplete: 'off',
      onInput: (e: Event) => {
        typed = (e.target as HTMLInputElement).value;
        updateConfirmBtn();
      },
    });

    field.appendChild(tokenInput);
    field.appendChild(renderCheckbox({
      label: 'I understand the consequences',
      checked: strongChecked,
      onChange: (checked) => {
        strongChecked = checked;
        updateConfirmBtn();
      },
    }));
    body.push(field);
  }

  const isVertical = opts.layout === 'vertical' || (opts.layout === undefined && (isDestructive || isStrong));

  const cancelBtn = el('button', {
    type: 'button',
    className: isVertical ? 'btn btn--ghost' : 'btn btn--secondary',
    onClick: () => opts.onCancel(),
  }, 'Cancel');

  confirmBtn = el('button', {
    type: 'button',
    className: (isDestructive || isStrong ? 'btn btn--danger' : 'btn btn--primary') + (isVertical ? ' btn--block' : ''),
    onClick: () => {
      if (!confirmBtn?.disabled) opts.onConfirm();
    },
  }, 'Confirm') as HTMLButtonElement;

  updateConfirmBtn();

  const modal = mountModal({
    title: opts.title || defaultTitle,
    description: opts.description,
    onClose: () => opts.onCancel(),
    body,
    actions: isVertical ? [confirmBtn, cancelBtn] : [cancelBtn, confirmBtn],
    footerClass: isVertical ? 'modal__footer--vertical' : undefined,
  });

  return modal;
}
