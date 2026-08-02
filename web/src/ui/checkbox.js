import { el } from '../lib/dom.js';

/**
 * Render a custom-styled checkbox with an optional label.
 *
 * @param {{
 *   checked: boolean,
 *   disabled?: boolean,
 *   onChange: (checked: boolean) => void,
 *   label?: string,
 *   id?: string,
 *   className?: string,
 * }} opts
 * @returns {HTMLLabelElement}
 */
export function renderCheckbox(opts) {
  const id = opts.id ?? `check-${Math.random().toString(36).slice(2, 9)}`;

  const input = el('input', {
    type: 'checkbox',
    id,
    className: 'check__native',
    checked: opts.checked,
    disabled: opts.disabled,
    onChange: (e) => {
      if (!opts.disabled) opts.onChange(e.target.checked);
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
  );
}
