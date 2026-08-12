import { el } from '../lib/dom.js';
import { mountModal, type ModalHandle } from './modal.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { renderButton } from './button.js';

export type ApiKeyOnceModalOpts = {
  name: string;
  rawKey: string;
  expiresAt?: string;
};

/**
 * Show one-time API key modal (raw_key never stored client-side after close).
 */
export function showApiKeyOnceModal(opts: ApiKeyOnceModalOpts): { destroy: () => void } {
  let modal: ModalHandle | null = null;

  function close(): void {
    modal?.destroy();
    modal = null;
  }

  function copyKey(): void {
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
          onFocus: (e: Event) => (e.target as HTMLInputElement).select(),
        }),
      ),
      opts.expiresAt
        ? el('p', { className: 'text-muted text-sm' }, `Expires: ${opts.expiresAt}`)
        : null,
    ],
    actions: [
      renderButton({
        label: 'Copy to clipboard',
        variant: 'secondary',
        onClick: copyKey,
      }),
      renderButton({
        label: 'I saved the key',
        variant: 'primary',
        onClick: close,
      }),
    ],
  });

  return { destroy: close };
}
