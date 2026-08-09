import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';

/**
 * Block the admin shell until the current EULA is accepted.
 *
 * @param {HTMLElement} root
 * @param {{ version: string, text: string }} eula
 * @returns {Promise<boolean>}
 */
export async function mountEulaGate(root, eula) {
  return new Promise((resolve) => {
    let destroyed = false;
    let loading = false;
    let error = null;
    let checked = false;

    function render() {
      if (destroyed) return;
      replaceChildren(root,
        el('div', { className: 'login-page' },
          el('div', { className: 'login-box login-box--narrow' },
            el('h1', { className: 'login-box__title' }, 'License agreement'),
            el('p', { className: 'login-box__sub' }, `Version ${eula.version}`),
            el('pre', {
              className: 'login-box__eula text-sm',
              style: { maxHeight: '240px', overflow: 'auto', whiteSpace: 'pre-wrap' },
            }, eula.text || ''),
            error
              ? el('div', { className: 'text-danger text-sm mb-3' }, error)
              : null,
            el('form', { onSubmit: handleSubmit },
              el('label', { className: 'form-check mb-3' },
                el('input', {
                  type: 'checkbox',
                  checked,
                  onChange: (e) => {
                    checked = e.target.checked;
                    render();
                  },
                }),
                ' I accept the BidShard on-premise license agreement',
              ),
              el('div', { className: 'form-actions' },
                el('button', {
                  type: 'submit',
                  className: 'btn btn--primary btn--block',
                  disabled: loading || !checked,
                }, loading ? 'Saving…' : 'Continue'),
              ),
            ),
          ),
        ),
      );
    }

    async function handleSubmit(e) {
      e.preventDefault();
      if (!checked) return;
      loading = true;
      error = null;
      render();
      const [, err] = await to(api('/api/v1/eula/accept', {
        method: 'POST',
        body: JSON.stringify({ version: eula.version }),
      }));
      loading = false;
      if (err) {
        error = err.message || 'Failed to record acceptance';
        render();
        return;
      }
      destroyed = true;
      root.replaceChildren();
      resolve(true);
    }

    render();
  });
}
