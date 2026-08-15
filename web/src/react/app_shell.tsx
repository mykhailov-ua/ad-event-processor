import { memo, useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { BrowserRouter, useLocation, useNavigate } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { startOpsOutboxBadge } from '../helpers/ops_outbox_badge.js';
import { startRUMCollector } from '../helpers/rum_collector.js';
import { memoryWatchOnRouteLeave } from '../helpers/memory_watch.js';
import { setSpaNavigate } from '../helpers/spa_navigate.js';
import { probeRouteChange } from '../helpers/perf_probe.js';
import { installCommandPalette } from '../ui/command_palette.js';
import { installConfirmHost } from '../ui/confirm_host.js';
import { installCustomScrollbars } from '../ui/custom_scrollbars.js';
import { renderIdempotencyRecoveryBanner } from '../ui/idempotency_banner.js';
import { mountShell } from '../ui/shell.js';
import { installSidebarSearch } from '../ui/sidebar_search.js';
import { installToastStack } from '../ui/toast_stack.js';
import { installErrorSurface } from '../lib/error_surface.js';
import { AppRoutes } from './app_routes.js';

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

const ImperativeShellHost = memo(function ImperativeShellHost() {
  const hostRef = useRef<HTMLDivElement>(null);
  const [outlet, setOutlet] = useState<HTMLElement | null>(null);

  useEffect(() => {
    const host = hostRef.current;
    const root = document.getElementById('root');
    if (!host || !root) return undefined;

    installErrorSurface(root);
    installConfirmHost(root);
    const toast = installToastStack(root);
    const scrollbars = installCustomScrollbars();

    const outletEl = document.createElement('div');
    outletEl.id = 'app-outlet';
    const idemBanner = renderIdempotencyRecoveryBanner();
    const shell = mountShell({
      outlet: outletEl,
      extraBanners: idemBanner ? [idemBanner] : [],
    });
    host.appendChild(shell.node);

    const sidebarSearch = installSidebarSearch(shell.searchAnchor);
    const cmdPalette = installCommandPalette({ focusSearch: (q) => sidebarSearch.focus(q) });

    const perms = auth.getUser()?.permissions ?? [];
    let opsBadgeFeed: { destroy?: () => void } | null = null;
    if (perms.includes('shards:read') && typeof shell.setOpsOutboxPending === 'function') {
      opsBadgeFeed = startOpsOutboxBadge(shell.setOpsOutboxPending);
    }

    const rum = startRUMCollector();
    setOutlet(outletEl);

    return () => {
      opsBadgeFeed?.destroy?.();
      rum.stop();
      cmdPalette.destroy();
      sidebarSearch.destroy();
      scrollbars.destroy();
      toast.destroy();
      shell.destroy();
      shell.node.remove();
      setOutlet(null);
    };
  }, []);

  return (
    <>
      <div ref={hostRef} />
      {outlet ? createPortal(<AppRoutes />, outlet) : null}
    </>
  );
});

/**
 * Authenticated admin chrome with React Router outlet inside the legacy shell.
 */
export function AppShell() {
  return (
    <BrowserRouter>
      <SpaLinkInterceptor />
      <RouteChangeEmitter />
      <ImperativeShellHost />
    </BrowserRouter>
  );
}
