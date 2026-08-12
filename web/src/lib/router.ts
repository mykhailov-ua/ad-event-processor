import { el } from './dom.js';
import * as auth from '../helpers/auth.js';
import { canAccessRoute } from '../helpers/route_guard.js';
import { probeRouteChange } from '../helpers/perf_probe.js';
import { memoryWatchOnRouteLeave } from '../helpers/memory_watch.js';
import type { RouteContext, RouteDef, ViewHandle, ViewModule } from './router_types.js';

export type { RouteContext, RouteDef, ViewHandle, ViewModule } from './router_types.js';

let routes: RouteDef[] = [];
let onNavigate: ((path: string) => void) | null = null;
let activeView: ViewHandle | null = null;
let outlet: HTMLElement | null = null;
/** Monotonic id so stale async navigations do not clobber a newer route. */
let routeGen = 0;

/**
 * Register application routes.
 */
export function configureRoutes(defs: RouteDef[]): void {
  routes = defs;
}

/**
 * Override navigation (pushState) handler.
 */
export function setNavigateHandler(fn: (path: string) => void): void {
  onNavigate = fn;
}

/**
 * Set the DOM outlet for view mounts.
 */
export function setOutlet(el: HTMLElement): void {
  outlet = el;
}

/**
 * Match a pathname against registered routes.
 */
export function matchRoute(
  pathname: string,
): { route: RouteDef; params: Record<string, string> } | null {
  const path = pathname.split('?')[0].replace(/\/$/, '') || '/';
  for (const route of routes) {
    const patternParts = route.path.split('/');
    const pathParts = path.split('/');
    if (patternParts.length !== pathParts.length) continue;
    const params: Record<string, string> = {};
    let ok = true;
    for (let i = 0; i < patternParts.length; i += 1) {
      const part = patternParts[i];
      const seg = pathParts[i];
      if (part.startsWith(':')) params[part.slice(1)] = seg;
      else if (part !== seg) ok = false;
    }
    if (ok) return { route, params };
  }
  return null;
}

/**
 * Navigate to a path and render.
 */
export function navigate(path: string): void {
  if (onNavigate) onNavigate(path);
  else window.history.pushState(null, '', path);
  void renderCurrent();
}

/**
 * Render the view for the current (or given) pathname.
 */
export async function renderCurrent(pathname = window.location.pathname): Promise<void> {
  if (!outlet) return;
  memoryWatchOnRouteLeave();
  const matched = matchRoute(pathname);
  const route = matched?.route;
  if (!route) {
    const mod = await import('../views/not_found.js') as ViewModule;
    if (activeView?.destroy) activeView.destroy();
    activeView = null;
    outlet.replaceChildren();
    activeView = mod.mount(outlet, {
      params: {},
      query: new URLSearchParams(window.location.search),
      navigate,
    }) ?? null;
    return;
  }

  const query = new URLSearchParams(window.location.search);
  const user = auth.getUser();
  const permissions = user?.permissions ?? [];
  const role = user?.role ?? '';
  if (!canAccessRoute(pathname, permissions, role)) {
    const mod = await import('../views/forbidden.js') as ViewModule;
    if (activeView?.destroy) activeView.destroy();
    activeView = null;
    outlet.replaceChildren();
    activeView = mod.mount(outlet, { params: {}, query, navigate }) ?? null;
    return;
  }

  const ctx: RouteContext = {
    params: matched!.params,
    query,
    navigate,
  };

  const loadGen = ++routeGen;
  const loadingTimer = window.setTimeout(() => {
    if (routeGen !== loadGen) return;
    replaceOutlet(el('span', { className: 'text-muted' }, 'Loading…'));
  }, 120);

  try {
    const mod = await route.load();
    window.clearTimeout(loadingTimer);
    if (routeGen !== loadGen) return;
    if (activeView?.destroy) activeView.destroy();
    activeView = null;
    outlet.replaceChildren();
    activeView = mod.mount(outlet, ctx) ?? null;
    probeRouteChange(pathname.split('?')[0] || '/');
    window.dispatchEvent(new Event('routechange'));
  } catch (err: unknown) {
    window.clearTimeout(loadingTimer);
    if (routeGen !== loadGen) return;
    const message = err instanceof Error ? err.message : String(err);
    replaceOutlet(
      el('div', { className: 'error-page' },
        el('div', { className: 'error-page__title' }, 'Failed to load view'),
        el('div', { className: 'text-muted' }, message),
      ),
    );
  }
}

function replaceOutlet(node: HTMLElement): void {
  if (!outlet) return;
  if (activeView?.destroy) activeView.destroy();
  activeView = null;
  outlet.replaceChildren(node);
}

/**
 * Wire popstate + click interception and render the initial route.
 */
export function startRouter(root: HTMLElement): void {
  setOutlet(root);
  window.addEventListener('popstate', () => {
    void renderCurrent();
  });
  document.addEventListener('click', (e) => {
    const target = e.target;
    const a = target instanceof Element ? target.closest('a[href]') : null;
    if (!(a instanceof HTMLAnchorElement)) return;
    if (a.target === '_blank' || a.hasAttribute('download')) return;
    const href = a.getAttribute('href');
    if (!href || !href.startsWith('/') || href.startsWith('//')) return;
    e.preventDefault();
    navigate(href);
  });
  void renderCurrent();
}
