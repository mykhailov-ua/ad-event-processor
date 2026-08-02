import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { apiConfirmed } from '../helpers/confirmed_api.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { mapServiceError } from '../helpers/service_error.js';

/**
 * Mount the platform bootstrap registration form.
 *
 * @param {HTMLElement} container
 * @param {{ navigate: (path: string) => void }} _ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container, _ctx) {
  let destroyed = false;
  const state = {
    installToken: '',
    email: '',
    password: '',
    trackingDomain: '',
    loading: false,
    error: null,
  };

  function render() {
    if (destroyed) return;
    replaceChildren(container,
      el('div', { className: 'login-page' },
        el('div', { className: 'login-box', style: { maxWidth: 420 } },
          el('h1', { className: 'login-box__title' }, 'Bootstrap'),
          el('p', { className: 'login-box__sub' }, 'Platform bootstrap'),
          state.error
            ? el('div', { style: { color: 'var(--error)', fontSize: 13, marginBottom: 12 } }, state.error)
            : null,
          el('form', { onSubmit: handleSubmit },
            el('div', { className: 'form-field' },
              el('label', { className: 'form-label', htmlFor: 'bootstrap-install-token' }, 'Install token'),
              el('input', {
                id: 'bootstrap-install-token',
                type: 'password',
                className: 'form-input',
                required: true,
                value: state.installToken,
                onInput: (e) => { state.installToken = e.target.value; },
              }),
            ),
            el('div', { className: 'form-field' },
              el('label', { className: 'form-label', htmlFor: 'bootstrap-admin-email' }, 'Admin email'),
              el('input', {
                id: 'bootstrap-admin-email',
                type: 'email',
                className: 'form-input',
                required: true,
                value: state.email,
                onInput: (e) => { state.email = e.target.value; },
              }),
            ),
            el('div', { className: 'form-field' },
              el('label', { className: 'form-label', htmlFor: 'bootstrap-admin-password' }, 'Admin password'),
              el('input', {
                id: 'bootstrap-admin-password',
                type: 'password',
                className: 'form-input',
                required: true,
                value: state.password,
                onInput: (e) => { state.password = e.target.value; },
              }),
            ),
            el('div', { className: 'form-field' },
              el('label', { className: 'form-label', htmlFor: 'bootstrap-tracking-domain' }, 'Tracking domain'),
              el('input', {
                id: 'bootstrap-tracking-domain',
                className: 'form-input',
                required: true,
                value: state.trackingDomain,
                onInput: (e) => { state.trackingDomain = e.target.value; },
              }),
            ),
            el('div', { className: 'form-actions' },
              el('button', {
                type: 'submit',
                className: 'btn btn--primary btn--block',
                disabled: state.loading,
              }, state.loading ? 'Initializing…' : 'Initialize platform'),
            ),
          ),
        ),
      ),
    );
  }

  async function handleSubmit(e) {
    e.preventDefault();
    state.loading = true;
    state.error = null;
    render();
    const [, err] = await to(apiConfirmed('/api/v1/settings/platform/bootstrap', {
      method: 'POST',
      headers: {
        'X-Install-Token': state.installToken,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        config: {
          tracking_domain: state.trackingDomain,
          default_currency: 'USD',
          timezone: 'UTC',
          ingress_schema: 'espx_native',
          profile: 'single_vps',
          network_interface: 'eth0',
          telemetry_enabled: true,
          edge_xdp: false,
          stripe: { enabled: false },
        },
        admin_email: state.email,
        admin_password: state.password,
      }),
    }));
    if (err) {
      if (err instanceof ConfirmCancelledError) {
        state.loading = false;
        render();
        return;
      }
      const view = mapServiceError(err);
      state.error = view.message;
      state.loading = false;
      render();
      return;
    }
    window.location.assign('/login');
  }

  render();

  return {
    destroy() {
      destroyed = true;
    },
  };
}
