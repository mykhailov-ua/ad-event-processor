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
import { installCustomScrollbars } from '../ui/custom_scrollbars.js';
import { mountShell } from '../ui/shell.js';
import { installCommandPalette } from '../ui/command_palette.js';
import { installSidebarSearch } from '../ui/sidebar_search.js';
import { startRUMCollector } from '../helpers/rum_collector.js';
import { startOpsOutboxBadge } from '../helpers/ops_outbox_badge.js';
import { renderIdempotencyRecoveryBanner } from '../ui/idempotency_banner.js';
import { syncDevModeAttribute } from '../helpers/dev_mode.js';
import { mountEulaGate } from '../ui/eula_gate.js';
import type { RouteContext, ViewHandle, ViewModule } from './router_types.js';

export type BootHandle = {
  destroy: () => void;
};

/**
 * Boot the authenticated admin SPA into root.
 */
export async function bootApp(root: HTMLElement): Promise<BootHandle | void> {
  if (window.location.pathname === '/bootstrap' || window.location.pathname === '/install/done') {
    await bootStandalone(root);
    return;
  }

  document.documentElement.setAttribute('data-theme', storage.getTheme());
  syncDevModeAttribute();
  installErrorSurface(root);
  installConfirmHost(root);
  const toast = installToastStack(root);
  const scrollbars = installCustomScrollbars();

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
  const me = meRes.data as {
    id: string;
    email: string;
    role: string;
    customer_id?: string;
    permissions?: string[];
  };
  auth.setUser({
    id: me.id,
    email: me.email,
    role: me.role,
    customer_id: me.customer_id ?? '',
    permissions: me.permissions ?? [],
  });

  loading.remove();

  const [eulaRes, eulaErr] = await to(api('/api/v1/eula'));
  if (!eulaErr && eulaRes?.data) {
    const eula = eulaRes.data as { required?: boolean; version?: string; text?: string };
    if (eula.required) {
      const accepted = await mountEulaGate(root, {
        version: eula.version ?? '',
        text: eula.text ?? '',
      });
      if (!accepted) return;
    }
  }

  const outlet = el('div', { id: 'app-outlet' });
  const idemBanner = renderIdempotencyRecoveryBanner();
  const shell = mountShell({
    outlet,
    extraBanners: idemBanner ? [idemBanner] : [],
  }) as {
    node: HTMLElement;
    searchAnchor: HTMLElement;
    destroy: () => void;
    setOpsOutboxPending?: (n: number) => void;
  };
  const sidebarSearch = installSidebarSearch(shell.searchAnchor);
  const cmdPalette = installCommandPalette({ focusSearch: (q) => sidebarSearch.focus(q) });
  root.appendChild(shell.node);

  configureRoutes(APP_ROUTES);
  setOutlet(outlet);
  startRouter(outlet);

  const rum = startRUMCollector();

  const perms = auth.getUser()?.permissions ?? [];
  let opsBadgeFeed: { destroy?: () => void } | null = null;
  if (perms.includes('shards:read') && typeof shell.setOpsOutboxPending === 'function') {
    opsBadgeFeed = startOpsOutboxBadge(shell.setOpsOutboxPending);
  }

  return {
    destroy() {
      opsBadgeFeed?.destroy?.();
      rum.stop();
      cmdPalette.destroy();
      sidebarSearch.destroy();
      scrollbars.destroy();
      toast.destroy();
      shell.destroy();
    },
  };
}

/**
 * Boot the login shell.
 */
export async function bootLogin(root: HTMLElement): Promise<void> {
  document.documentElement.setAttribute('data-theme', storage.getTheme());

  const [metaRes, metaErr] = await to(fetch('/api/v1/meta', { credentials: 'same-origin' }).then(async (res) => {
    if (!res.ok) throw new Error('meta unavailable');
    const data = await res.json() as { bootstrap_complete?: boolean };
    return { data };
  }));
  if (!metaErr && metaRes?.data?.bootstrap_complete === false) {
    window.location.assign('/bootstrap');
    return;
  }

  installToastStack(root);
  installCustomScrollbars();
  const outlet = el('div', { id: 'login-outlet', className: 'login-root' });
  root.appendChild(outlet);
  void import('../views/login.js').then((mod) => {
    const view = mod as ViewModule;
    view.mount(outlet, {
      params: {},
      query: new URLSearchParams(window.location.search),
      navigate: (path: string) => window.location.assign(path),
    });
  });
}

/**
 * Boot bootstrap / install-done standalone routes without the main shell.
 */
export async function bootStandalone(root: HTMLElement): Promise<void> {
  document.documentElement.setAttribute('data-theme', storage.getTheme());
  installErrorSurface(root);
  installConfirmHost(root);
  installToastStack(root);

  const outlet = el('div', { id: 'login-outlet', className: 'login-root' });
  root.appendChild(outlet);

  const path = window.location.pathname;
  const route = APP_ROUTES.find((r) => r.path === path);
  const ctx: RouteContext = {
    params: {},
    query: new URLSearchParams(window.location.search),
    navigate: (p: string) => window.location.assign(p),
  };
  if (!route) {
    const mod = await import('../views/not_found.js') as ViewModule;
    mod.mount(outlet, ctx);
    return;
  }
  const mod = await route.load();
  mod.mount(outlet, ctx);
}

export type { ViewHandle };
