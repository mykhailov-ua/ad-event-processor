import { el } from '../lib/dom.js';
import { mapServiceError } from '../helpers/service_error.js';
import { renderIcon } from './icon.js';

/**
 * @param {unknown} error
 * @param {string} [fallbackTitle]
 */
export function renderErrorBlock(error, fallbackTitle = 'Error') {
  if (!error) return el('div');
  const view = mapServiceError(error);
  return el('div', { className: 'error-page', style: { padding: '32px 0', textAlign: 'center' } },
    renderIcon('alert-triangle', { size: 36, className: 'error-page__icon text-muted mb-3' }),
    el('div', { className: 'error-page__code' }, String(view.status ?? '??')),
    el('div', { className: 'error-page__title' }, view.title || fallbackTitle),
    el('div', { className: 'error-page__desc text-muted' }, view.message),
    view.code && view.code !== view.message
      ? el('div', { className: 'text-muted', style: { fontSize: 11, marginTop: 8 } }, view.code)
      : null,
  );
}
