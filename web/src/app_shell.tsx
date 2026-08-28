import { lazy, Suspense, useEffect, type ReactNode } from 'react';
import { BrowserRouter, useNavigate } from 'react-router-dom';
import { setSpaNavigate } from './helpers/spa_navigate.js';
import { AppRoutes } from './app_routes.js';

const ShellLayout = lazy(() =>
  import('./ui/shell/app_shell_layout.js').then((mod) => ({
    default: mod.AppShellLayout,
  }))
);

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

function RouteFallback() {
  return <span className="text-muted">Loading...</span>;
}

export function AppShell() {
  return (
    <BrowserRouter>
      <SpaLinkInterceptor />
      <Suspense fallback={<RouteFallback />}>
        <ShellLayout>
          <AppRoutes />
        </ShellLayout>
      </Suspense>
    </BrowserRouter>
  );
}
