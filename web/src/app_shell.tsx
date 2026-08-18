import { lazy, Suspense, useEffect, type ReactNode } from 'react';
import { BrowserRouter, useLocation, useNavigate } from 'react-router-dom';
import { memoryWatchOnRouteLeave } from './helpers/memory_watch.js';
import { setSpaNavigate } from './helpers/spa_navigate.js';
import { probeRouteChange } from './helpers/perf_probe.js';
import { AppRoutes } from './app_routes.js';

const ShellLayout = lazy(() =>
  import('./components/shell_layout.js').then((mod) => ({
    default: mod.ShellLayout,
  }))
);

const SelfServeShellLayout = lazy(() =>
  import('./components/selfserve_shell_layout.js').then((mod) => ({
    default: mod.SelfServeShellLayout,
  }))
);

function LayoutSwitcher({ children }: { children: ReactNode }) {
  const location = useLocation();
  const selfServe =
    location.pathname === '/selfserve' || location.pathname.startsWith('/selfserve/');
  if (selfServe) {
    return (
      <Suspense fallback={<span className="text-muted">Loading…</span>}>
        <SelfServeShellLayout>{children}</SelfServeShellLayout>
      </Suspense>
    );
  }
  return (
    <Suspense fallback={<span className="text-muted">Loading…</span>}>
      <ShellLayout>{children}</ShellLayout>
    </Suspense>
  );
}

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

export function AppShell() {
  return (
    <BrowserRouter>
      <SpaLinkInterceptor />
      <RouteChangeEmitter />
      <LayoutSwitcher>
        <AppRoutes />
      </LayoutSwitcher>
    </BrowserRouter>
  );
}
