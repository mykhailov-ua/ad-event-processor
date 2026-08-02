import { el } from '../lib/dom.js';

/**
 * Render a labeled form field with optional hint and error message.
 *
 * @param {{
 *   label: string,
 *   htmlFor?: string,
 *   error?: string|null,
 *   hint?: string,
 *   children: HTMLElement,
 * }} opts
 * @returns {HTMLElement}
 */
export function renderFormField(opts) {
  const field = el('div', {
    className: [
      'form-field',
      opts.error ? 'form-field--error' : '',
    ].filter(Boolean).join(' '),
  });

  field.appendChild(el('label', {
    className: 'form-label',
    htmlFor: opts.htmlFor,
  }, opts.label));

  field.appendChild(opts.children);

  if (opts.hint && !opts.error) {
    field.appendChild(el('p', { className: 'form-hint' }, opts.hint));
  }

  if (opts.error) {
    field.appendChild(el('div', { className: 'form-error', role: 'alert' }, opts.error));
  }

  return field;
}
