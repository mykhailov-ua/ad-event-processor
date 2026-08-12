import { el } from '../lib/dom.js';
import { mapServiceError } from '../helpers/service_error.js';
import { renderIcon } from './icon.js';

/**
 * Render a centered page-level error block from an API or network error.
 */
export function renderErrorBlock(error: unknown, fallbackTitle = 'Error'): HTMLElement {
  if (!error) return el('div');
  const view = mapServiceError(error);
  return el('div', { className: 'error-page' },
    renderIcon('alert-triangle', { size: 36, className: 'error-page__icon text-muted mb-3' }),
    el('div', { className: 'error-page__code' }, String(view.status ?? '??')),
    el('div', { className: 'error-page__title' }, view.title || fallbackTitle),
    el('div', { className: 'error-page__desc text-muted' }, view.message),
    view.code && view.code !== view.message
      ? el('div', { className: 'text-muted text-xs mt-2' }, view.code)
      : null,
  );
}
