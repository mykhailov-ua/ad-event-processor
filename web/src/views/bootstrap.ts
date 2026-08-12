import { el, replaceChildren, eventTargetValue, eventTargetChecked } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { apiConfirmed } from '../helpers/confirmed_api.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import type { RouteContext, ViewHandle } from '../lib/router_types.js';
import { renderButton } from '../ui/button.js';

type BootstrapState = {
  installToken: string;
  email: string;
  password: string;
  trackingDomain: string;
  eulaText: string;
  eulaVersion: string;
  eulaAccepted: boolean;
  loading: boolean;
  error: string | null;
};

type EulaPayload = {
  version?: string;
  text?: string;
};

type BootstrapResponse = {
  click_url_template?: string;
};



/**
 * Mount the platform bootstrap registration form.
 */
export function mount(container: HTMLElement, _ctx: RouteContext): ViewHandle {
  let destroyed = false;
  const state: BootstrapState = {
    installToken: '',
    email: '',
    password: '',
    trackingDomain: '',
    eulaText: '',
    eulaVersion: '',
    eulaAccepted: false,
    loading: false,
    error: null,
  };

  async function loadEula(): Promise<void> {
    const [res, err] = await to(fetch('/api/v1/eula', { credentials: 'same-origin' }).then(async (r) => {
      if (!r.ok) throw new Error('eula unavailable');
      return { data: await r.json() as EulaPayload };
    }));
    if (err || !res?.data) return;
    state.eulaVersion = res.data.version || '';
    state.eulaText = res.data.text || '';
    render();
  }

  function render(): void {
    if (destroyed) return;
    replaceChildren(container,
      el('div', { className: 'login-page' },
        el('div', { className: 'login-box login-box--narrow' },
          el('h1', { className: 'login-box__title' }, 'Bootstrap'),
          el('p', { className: 'login-box__sub' }, 'Platform bootstrap'),
          state.error
            ? el('div', { className: 'text-danger text-sm mb-3' }, state.error)
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
                onInput: (e: Event) => { state.installToken = eventTargetValue(e); },
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
                onInput: (e: Event) => { state.email = eventTargetValue(e); },
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
                onInput: (e: Event) => { state.password = eventTargetValue(e); },
              }),
            ),
            el('div', { className: 'form-field' },
              el('label', { className: 'form-label', htmlFor: 'bootstrap-tracking-domain' }, 'Tracking domain'),
              el('input', {
                id: 'bootstrap-tracking-domain',
                className: 'form-input',
                required: true,
                value: state.trackingDomain,
                onInput: (e: Event) => { state.trackingDomain = eventTargetValue(e); },
              }),
            ),
            state.eulaText
              ? el('div', { className: 'form-field' },
                el('pre', {
                  className: 'text-sm',
                  style: { maxHeight: '160px', overflow: 'auto', whiteSpace: 'pre-wrap' },
                }, state.eulaText),
                el('label', { className: 'form-check' },
                  el('input', {
                    type: 'checkbox',
                    required: true,
                    checked: state.eulaAccepted,
                    onChange: (e: Event) => { state.eulaAccepted = eventTargetChecked(e); },
                  }),
                  ' I accept the on-premise license agreement',
                ),
              )
              : null,
            el('div', { className: 'form-actions' },
              renderButton({
                label: state.loading ? 'Initializing…' : 'Initialize platform',
                variant: 'primary',
                type: 'submit',
                className: 'btn--block',
                loading: state.loading,
                disabled: state.loading,
              }),
            ),
          ),
        ),
      ),
    );
  }

  async function handleSubmit(e: Event): Promise<void> {
    e.preventDefault();
    state.loading = true;
    state.error = null;
    render();
    const [res, err] = await to(apiConfirmed('/api/v1/settings/platform/bootstrap', {
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
          ingress_schema: 'ad_event_processor_native',
          profile: 'single_vps',
          network_interface: 'eth0',
          telemetry_enabled: true,
          edge_xdp: false,
          edge_expose_click: true,
          edge_expose_openrtb: false,
          stripe: { enabled: false },
        },
        admin_email: state.email,
        admin_password: state.password,
        eula_version: state.eulaAccepted ? state.eulaVersion : undefined,
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
    sessionStorage.setItem('install_tracking_domain', state.trackingDomain);
    const data = res?.data as BootstrapResponse | null | undefined;
    if (data?.click_url_template) {
      sessionStorage.setItem('install_click_url', data.click_url_template);
    }
    sessionStorage.setItem('install_ingress_enabled', '0');
    window.location.assign('/install/done');
  }

  render();
  loadEula();

  return {
    destroy() {
      destroyed = true;
    },
  };
}
