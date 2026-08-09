import { el } from './dom.js';
import * as auth from '../helpers/auth.js';
import { canAccessRoute } from '../helpers/route_guard.js';
import { probeRouteChange } from '../helpers/perf_probe.js';
import { memoryWatchOnRouteLeave } from '../helpers/memory_watch.js';

/** @typedef {{ path: string, shell?: boolean, load: () => Promise<{ mount: (el: HTMLElement, ctx: RouteContext) => ViewHandle }> }} RouteDef */

/** @typedef {{ params: Record<string, string>, query: URLSearchParams, navigate: (path: string) => void }} RouteContext */

/** @typedef {{ destroy?: () => void }} ViewHandle */

let routes = [];
/** @type {((path: string) => void)|null} */
let onNavigate = null;
/** @type {ViewHandle|null} */
let activeView = null;
/** @type {HTMLElement|null} */
let outlet = null;
/** Monotonic id so stale async navigations do not clobber a newer route. */
let routeGen = 0;

/**
 * @param {RouteDef[]} defs
 */
export function configureRoutes(defs) {
  routes = defs;
}

/**
 * @param {(path: string) => void} fn
 */
export function setNavigateHandler(fn) {
  onNavigate = fn;
}

/**
 * @param {HTMLElement} el
 */
export function setOutlet(el) {
  outlet = el;
}

/**
 * @param {string} pathname
 * @returns {{ route: RouteDef, params: Record<string, string> }|null}
 */
export function matchRoute(pathname) {
  const path = pathname.split('?')[0].replace(/\/$/, '') || '/';
  for (const route of routes) {
    const patternParts = route.path.split('/');
    const pathParts = path.split('/');
    if (patternParts.length !== pathParts.length) continue;
    /** @type {Record<string, string>} */
    const params = {};
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
 * @param {string} path
 */
export function navigate(path) {
  if (onNavigate) onNavigate(path);
  else window.history.pushState(null, '', path);
  renderCurrent();
}

/**
 * @param {string} [pathname]
 */
export async function renderCurrent(pathname = window.location.pathname) {
  if (!outlet) return;
  memoryWatchOnRouteLeave();
  const matched = matchRoute(pathname);
  const route = matched?.route;
  if (!route) {
    const mod = await import('../views/not_found.js');
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
    const mod = await import('../views/forbidden.js');
    if (activeView?.destroy) activeView.destroy();
    activeView = null;
    outlet.replaceChildren();
    activeView = mod.mount(outlet, { params: {}, query, navigate }) ?? null;
    return;
  }

  const ctx = {
    params: matched.params,
    query,
    navigate,
  };

  // Keep current view until the next module is ready; only flash Loading if import is slow.
  const loadGen = ++routeGen;
  let loadingTimer = window.setTimeout(() => {
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
  } catch (err) {
    window.clearTimeout(loadingTimer);
    if (routeGen !== loadGen) return;
    replaceOutlet(
      el('div', { className: 'error-page' },
        el('div', { className: 'error-page__title' }, 'Failed to load view'),
        el('div', { className: 'text-muted' }, err?.message ?? String(err)),
      ),
    );
  }
}

/**
 * @param {HTMLElement} node
 */
function replaceOutlet(node) {
  if (!outlet) return;
  if (activeView?.destroy) activeView.destroy();
  activeView = null;
  outlet.replaceChildren(node);
}

/**
 * @param {HTMLElement} root
 */
export function startRouter(root) {
  setOutlet(root);
  window.addEventListener('popstate', () => renderCurrent());
  document.addEventListener('click', (e) => {
    const a = e.target instanceof Element ? e.target.closest('a[href]') : null;
    if (!a || a.target === '_blank' || a.hasAttribute('download')) return;
    const href = a.getAttribute('href');
    if (!href || !href.startsWith('/') || href.startsWith('//')) return;
    e.preventDefault();
    navigate(href);
  });
  renderCurrent();
}
