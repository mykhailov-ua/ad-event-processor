import * as storage from '../helpers/storage.js';
import * as auth from '../helpers/auth.js';
import { api } from '../helpers/api_client.js';
import { redirectToLogin } from '../helpers/session.js';
import { to } from './to.js';
import { el } from './dom.js';
import { configureRoutes, setOutlet, startRouter } from './router.js';
import { APP_ROUTES } from './routes.js';
import { installErrorSurface } from './error_surface.js';
import { installConfirmHost } from '../ui/confirm_host.js';
import { installToastStack } from '../ui/toast_stack.js';
import { mountShell } from '../ui/shell.js';
import { installCommandPalette } from '../ui/command_palette.js';
import { startRUMCollector } from '../helpers/rum_collector.js';
import { renderIdempotencyRecoveryBanner } from '../ui/idempotency_banner.js';
import { syncDevModeAttribute } from '../helpers/dev_mode.js';

/**
 * @param {HTMLElement} root
 */
export async function bootApp(root) {
  if (window.location.pathname === '/bootstrap') {
    await bootStandalone(root);
    return;
  }

  document.documentElement.setAttribute('data-theme', storage.getTheme());
  syncDevModeAttribute();
  installErrorSurface(root);
  installConfirmHost(root);
  const toast = installToastStack(root);

  auth.hydrateFromBoot();

  const loading = el('div', {
    className: 'main',
    style: { alignItems: 'center', justifyContent: 'center' },
  }, el('span', { className: 'text-muted' }, 'Loading…'));
  root.appendChild(loading);

  const [meRes, meErr] = await to(api('/api/v1/auth/me'));
  if (meErr) {
    redirectToLogin('session');
    return;
  }
  const csrf = meRes.headers.get('X-CSRF-Token');
  if (csrf) auth.setCsrfFromLoginResponse(csrf);
  auth.setUser({
    id: meRes.data.id,
    email: meRes.data.email,
    role: meRes.data.role,
    customer_id: meRes.data.customer_id,
    permissions: meRes.data.permissions ?? [],
  });

  loading.remove();

  const outlet = el('div', { id: 'app-outlet' });
  const idemBanner = renderIdempotencyRecoveryBanner();
  const cmdPalette = installCommandPalette();
  const shell = mountShell({
    outlet,
    extraBanners: idemBanner ? [idemBanner] : [],
    onOpenSearch: (query = '') => cmdPalette.open(query),
  });
  root.appendChild(shell.node);

  configureRoutes(APP_ROUTES);
  setOutlet(outlet);
  startRouter(outlet);

  const rum = startRUMCollector();

  return {
    destroy() {
      rum.stop();
      cmdPalette.destroy();
      toast.destroy();
      shell.destroy();
    },
  };
}

/**
 * @param {HTMLElement} root
 */
export async function bootLogin(root) {
  document.documentElement.setAttribute('data-theme', storage.getTheme());

  const [metaRes, metaErr] = await to(fetch('/api/v1/meta', { credentials: 'same-origin' }).then(async (res) => {
    if (!res.ok) throw new Error('meta unavailable');
    const data = await res.json();
    return { data };
  }));
  if (!metaErr && metaRes?.data?.bootstrap_complete === false) {
    window.location.assign('/bootstrap');
    return;
  }

  installToastStack(root);
  const outlet = el('div', { id: 'login-outlet', className: 'login-root' });
  root.appendChild(outlet);
  import('../views/login.js').then((mod) => {
    mod.mount(outlet, {
      params: {},
      query: new URLSearchParams(window.location.search),
      navigate: (path) => window.location.assign(path),
    });
  });
}

/**
 * @param {HTMLElement} root
 */
export async function bootStandalone(root) {
  document.documentElement.setAttribute('data-theme', storage.getTheme());
  installErrorSurface(root);
  installConfirmHost(root);
  installToastStack(root);

  const outlet = el('div', { id: 'login-outlet', className: 'login-root' });
  root.appendChild(outlet);

  const path = window.location.pathname;
  const route = APP_ROUTES.find((r) => r.path === path);
  if (!route) {
    const mod = await import('../views/not_found.js');
    mod.mount(outlet, {
      params: {},
      query: new URLSearchParams(window.location.search),
      navigate: (p) => window.location.assign(p),
    });
    return;
  }
  const mod = await route.load();
  mod.mount(outlet, {
    params: {},
    query: new URLSearchParams(window.location.search),
    navigate: (p) => window.location.assign(p),
  });
}
