import { el } from '../lib/dom.js';

export type CheckboxOpts = {
  checked: boolean;
  disabled?: boolean;
  onChange: (checked: boolean) => void;
  label?: string;
  id?: string;
  className?: string;
};

/**
 * Render a custom-styled checkbox with an optional label.
 */
export function renderCheckbox(opts: CheckboxOpts): HTMLLabelElement {
  const id = opts.id ?? `check-${Math.random().toString(36).slice(2, 9)}`;

  const input = el('input', {
    type: 'checkbox',
    id,
    className: 'check__native',
    checked: opts.checked,
    disabled: opts.disabled,
    onChange: (e: Event) => {
      const target = e.target as HTMLInputElement;
      if (!opts.disabled) opts.onChange(target.checked);
    },
  });

  const box = el('span', { className: 'check__box', 'aria-hidden': 'true' });

  const labelText = opts.label
    ? el('span', { className: 'check__label' }, opts.label)
    : null;

  return el('label', {
    className: [
      'check',
      opts.disabled ? 'check--disabled' : '',
      opts.className ?? '',
    ].filter(Boolean).join(' '),
  },
    input,
    box,
    labelText,
  ) as HTMLLabelElement;
}
