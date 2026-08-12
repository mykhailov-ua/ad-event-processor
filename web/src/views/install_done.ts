import { el, replaceChildren } from '../lib/dom.js';
import { renderButton } from '../ui/button.js';
import { to } from '../lib/to.js';
import type { RouteContext, ViewHandle } from '../lib/router_types.js';

type HealthStatus = 'pending' | 'pass' | 'fail';

type InstallState = {
  controlHealth: HealthStatus;
  bootstrapComplete: boolean;
  clickTemplate: string;
  trackingDomain: string;
  ingressEnabled: boolean;
  error: string | null;
};

type MetaPayload = {
  bootstrap_complete?: boolean;
};

/**
 * Mount the post-install checklist shown after platform bootstrap.
 */
export function mount(container: HTMLElement, ctx: RouteContext): ViewHandle {
  let destroyed = false;
  const state: InstallState = {
    controlHealth: 'pending',
    bootstrapComplete: false,
    clickTemplate: sessionStorage.getItem('install_click_url') || '',
    trackingDomain: sessionStorage.getItem('install_tracking_domain') || '',
    ingressEnabled: false,
    error: null,
  };

  /**
   * Build a checklist row with pass/fail/pending mark.
   */
  function checkRow(status: HealthStatus, label: string): HTMLElement {
    const mark = status === 'pass' ? '✓' : status === 'fail' ? '✗' : '…';
    return el('li', { className: 'install-check install-check--' + status }, `${mark} ${label}`);
  }

  function render(): void {
    if (destroyed) return;
    const clickExample = state.clickTemplate
      ? state.clickTemplate.replace('{campaign_id}', 'demo').replace('{sub1}', 'test')
      : (state.trackingDomain
        ? `https://${state.trackingDomain}/click?campaign_id=demo&sub1=test`
        : '');
    replaceChildren(container,
      el('div', { className: 'login-page' },
        el('div', { className: 'login-box login-box--narrow' },
          el('h1', { className: 'login-box__title' }, 'Install complete'),
          el('p', { className: 'login-box__sub' }, 'Verify these items before sending traffic.'),
          state.error
            ? el('div', { className: 'text-danger text-sm mb-3' }, state.error)
            : null,
          el('ul', { className: 'install-checklist' },
            checkRow(
              state.bootstrapComplete ? 'pass' : (state.controlHealth === 'fail' ? 'fail' : 'pending'),
              'Platform bootstrap saved',
            ),
            checkRow(
              state.controlHealth === 'pass' ? 'pass' : (state.controlHealth === 'fail' ? 'fail' : 'pending'),
              'Control API healthy',
            ),
            checkRow(
              state.trackingDomain ? 'pass' : 'pending',
              state.trackingDomain
                ? `DNS: point ${state.trackingDomain} A-record to this server`
                : 'DNS: set TRACKING_DOMAIN in settings',
            ),
            checkRow(
              state.ingressEnabled ? 'pass' : 'pending',
              state.ingressEnabled
                ? 'Ingress (Caddy) enabled — HTTPS on tracking/admin hosts'
                : 'Ingress optional — enable INGRESS_ENABLED=1 for automatic TLS',
            ),
          ),
          clickExample
            ? el('p', { className: 'text-sm mt-3' },
              'Sample click: ',
              el('a', { href: clickExample, className: 'link' }, clickExample),
            )
            : null,
          el('div', { className: 'form-actions mt-4' },
            renderButton({
              label: 'Continue to login',
              variant: 'primary',
              className: 'btn--block',
              onClick: () => ctx.navigate('/login'),
            }),
          ),
          el('p', { className: 'text-muted text-sm mt-3' },
            'Run on server: bash scripts/install/bidshard-install.sh doctor',
          ),
        ),
      ),
    );
  }

  async function loadStatus(): Promise<void> {
    const [, healthErr] = await to(fetch('/health', { credentials: 'same-origin' }));
    state.controlHealth = healthErr ? 'fail' : 'pass';

    const [metaRes, metaErr] = await to(fetch('/api/v1/meta', { credentials: 'same-origin' }).then(async (res) => {
      if (!res.ok) throw new Error('meta unavailable');
      return { data: await res.json() as MetaPayload };
    }));
    if (!metaErr && metaRes?.data) {
      state.bootstrapComplete = metaRes.data.bootstrap_complete === true;
    }

    state.ingressEnabled = new URLSearchParams(window.location.search).get('ingress') === '1'
      || sessionStorage.getItem('install_ingress_enabled') === '1';
    render();
  }

  render();
  loadStatus();

  return {
    destroy() {
      destroyed = true;
    },
  };
}
