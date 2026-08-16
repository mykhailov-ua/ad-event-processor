import { lazy, Suspense, useEffect } from 'react';
import { BrowserRouter, useLocation, useNavigate } from 'react-router-dom';
import { memoryWatchOnRouteLeave } from './helpers/memory_watch.js';
import { setSpaNavigate } from './helpers/spa_navigate.js';
import { probeRouteChange } from './helpers/perf_probe.js';
import { AppRoutes } from './app_routes.js';

const ShellLayout = lazy(() => import('./components/shell_layout.js').then((mod) => ({
  default: mod.ShellLayout,
})));

/**
 * Mirror legacy router link interception for plain anchor tags.
 */
function SpaLinkInterceptor() {
  const navigate = useNavigate();

  useEffect(() => {
    setSpaNavigate(navigate);
    const onClick = (e: MouseEvent) => {
      const target = e.target;
      const a = target instanceof Element ? target.closest('a[href]') : null;
      if (!(a instanceof HTMLAnchorElement)) return;
      if (a.target === '_blank' || a.hasAttribute('download')) return;
      const href = a.getAttribute('href');
      if (!href || !href.startsWith('/') || href.startsWith('//')) return;
      e.preventDefault();
      navigate(href);
    };
    document.addEventListener('click', onClick);
    return () => document.removeEventListener('click', onClick);
  }, [navigate]);

  return null;
}

/**
 * Emit routechange + perf/memory hooks consumed by shell chrome.
 */
function RouteChangeEmitter() {
  const location = useLocation();

  useEffect(() => {
    memoryWatchOnRouteLeave();
    const path = location.pathname.split('?')[0] || '/';
    probeRouteChange(path);
    window.dispatchEvent(new Event('routechange'));
  }, [location.pathname]);

  return null;
}

/**
 * Authenticated admin chrome with React Router outlet.
 */
export function AppShell() {
  return (
    <BrowserRouter>
      <SpaLinkInterceptor />
      <RouteChangeEmitter />
      <Suspense fallback={<span className="text-muted">Loading…</span>}>
        <ShellLayout>
          <AppRoutes />
        </ShellLayout>
      </Suspense>
    </BrowserRouter>
  );
}
