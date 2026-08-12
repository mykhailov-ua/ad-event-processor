import { el } from '../lib/dom.js';

export type FormFieldOpts = {
  label: string;
  htmlFor?: string;
  error?: string | null;
  hint?: string;
  /** Reserve one feedback line to avoid layout shift when errors appear. */
  reserveFeedback?: boolean;
  children: HTMLElement;
};

/**
 * Apply aria-invalid on common control types inside a form field.
 */
function markInvalid(control: HTMLElement, invalid: boolean): void {
  if (
    control instanceof HTMLInputElement
    || control instanceof HTMLTextAreaElement
    || control instanceof HTMLSelectElement
  ) {
    if (invalid) control.setAttribute('aria-invalid', 'true');
    else control.removeAttribute('aria-invalid');
  }
}

/**
 * Render a labeled form field with optional hint and error message.
 */
export function renderFormField(opts: FormFieldOpts): HTMLElement {
  const field = el('div', {
    className: [
      'form-field',
      opts.error ? 'form-field--error' : '',
      opts.reserveFeedback ? 'form-field--reserve-feedback' : '',
    ].filter(Boolean).join(' '),
  });

  field.appendChild(el('label', {
    className: 'form-label',
    htmlFor: opts.htmlFor,
  }, opts.label));

  markInvalid(opts.children, Boolean(opts.error));
  field.appendChild(opts.children);

  const feedback = el('div', { className: 'form-field__feedback' });
  if (opts.error) {
    feedback.appendChild(el('div', { className: 'form-error', role: 'alert' }, opts.error));
  } else if (opts.hint) {
    feedback.appendChild(el('p', { className: 'form-hint' }, opts.hint));
  }
  field.appendChild(feedback);

  return field;
}
