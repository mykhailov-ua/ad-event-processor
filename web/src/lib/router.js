import { el } from './dom.js';

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
  const ctx = {
    params: matched.params,
    query,
    navigate,
  };

  replaceOutlet(el('span', { className: 'text-muted' }, 'Loading…'));

  try {
    const mod = await route.load();
    if (activeView?.destroy) activeView.destroy();
    activeView = null;
    outlet.replaceChildren();
    activeView = mod.mount(outlet, ctx) ?? null;
  } catch (err) {
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
