import { el } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { createApiKey } from '../helpers/api_keys_api.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { showApiKeyOnceModal } from '../ui/api_key_once_modal.js';

/**
 * Mount self-serve API key creation block on customer detail.
 *
 * @param {HTMLElement} container
 * @param {{ canCreate: boolean }} opts
 * @returns {{ destroy: () => void }}
 */
export function mountCustomerApiKeysPanel(container, opts) {
  let destroyed = false;
  let name = '';
  let busy = false;
  let error = null;
  /** @type {{ destroy: () => void }|null} */
  let keyModal = null;

  async function createKey() {
    if (!opts.canCreate || busy) return;
    const trimmed = name.trim();
    if (!trimmed) {
      error = 'Key name is required';
      render();
      return;
    }
    busy = true;
    error = null;
    render();
    const [data, err] = await to(createApiKey(trimmed));
    if (destroyed) return;
    busy = false;
    if (err) {
      if (err instanceof ConfirmCancelledError) {
        render();
        return;
      }
      error = mapServiceError(err).message;
      render();
      return;
    }
    name = '';
    if (data?.raw_key) {
      keyModal?.destroy();
      keyModal = showApiKeyOnceModal({
        name: data.name ?? trimmed,
        rawKey: data.raw_key,
        expiresAt: data.expires_at,
      });
    } else {
      pushToastMessage({ title: 'Key created', message: 'No raw key returned by API' });
    }
    render();
  }

  function render() {
    container.replaceChildren(
      el('section', { className: 'section-block section-card stack' },
        el('h2', { className: 'subsection-title' }, 'API keys'),
        el('p', { className: 'text-muted text-sm' },
          'Create integration keys for tracking and automation. The secret is shown once after creation.',
        ),
        error ? el('p', { className: 'text-danger text-sm' }, error) : null,
        opts.canCreate
          ? el('div', { className: 'flex items-end gap-2 flex-wrap' },
            el('label', { className: 'form-field flex-1', htmlFor: 'api-key-name', style: { minWidth: '12rem' } },
              'Key name',
              el('input', {
                id: 'api-key-name',
                className: 'form-input',
                placeholder: 'e.g. Keitaro postback',
                value: name,
                disabled: busy,
                onInput: (e) => { name = e.target.value; },
              }),
            ),
            el('button', {
              type: 'button',
              className: 'btn btn--primary btn--sm',
              disabled: busy || !name.trim(),
              onClick: createKey,
            }, busy ? 'Creating…' : 'Create API key'),
          )
          : el('p', { className: 'text-muted text-sm' }, 'You need campaigns:write to create API keys.'),
      ),
    );
  }

  render();
  return {
    destroy() {
      destroyed = true;
      keyModal?.destroy();
    },
  };
}
