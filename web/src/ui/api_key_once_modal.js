import { el } from '../lib/dom.js';
import { mountModal } from './modal.js';
import { pushToastMessage } from '../helpers/toast_ui.js';

/**
 * Show one-time API key modal (raw_key never stored client-side after close).
 *
 * @param {{ name: string, rawKey: string, expiresAt?: string }} opts
 * @returns {{ destroy: () => void }}
 */
export function showApiKeyOnceModal(opts) {
  let modal = null;

  function close() {
    modal?.destroy();
    modal = null;
  }

  function copyKey() {
    navigator.clipboard?.writeText(opts.rawKey).then(() => {
      pushToastMessage({ title: 'Copied', message: 'API key copied to clipboard' });
    }).catch(() => {
      pushToastMessage({ title: 'Copy failed', message: 'Select the key and copy manually' });
    });
  }

  modal = mountModal({
    title: 'API key created',
    description: `Key "${opts.name}" is shown once. Store it securely — it cannot be retrieved later.`,
    onClose: close,
    body: [
      el('label', { className: 'form-field', htmlFor: 'api-key-raw' },
        'Secret key',
        el('input', {
          id: 'api-key-raw',
          className: 'form-input font-mono',
          readOnly: true,
          value: opts.rawKey,
          onFocus: (e) => e.target.select(),
        }),
      ),
      opts.expiresAt
        ? el('p', { className: 'text-muted text-sm' }, `Expires: ${opts.expiresAt}`)
        : null,
    ],
    actions: [
      el('button', {
        type: 'button',
        className: 'btn btn--secondary',
        onClick: copyKey,
      }, 'Copy to clipboard'),
      el('button', {
        type: 'button',
        className: 'btn btn--primary',
        onClick: close,
      }, 'I saved the key'),
    ],
  });

  return { destroy: close };
}
