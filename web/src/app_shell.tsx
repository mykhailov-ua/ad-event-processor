import { useMemo, useState } from 'react';
import { NavLink, Outlet } from 'react-router-dom';

import { logout } from '@/api/auth_api';
import {
  TrackerShellHeaderActions,
  TrackerShellHeaderSearch,
  TrackerShellSidebarToggle,
} from '@/shell/tracker_shell_header';
import { Button } from '@/components/ui/button';
import { AdminDevBanner } from '@/shell/admin_dev_banner';
import { AppErrorBoundary } from '@/shell/app_error_boundary';
import { BreadcrumbProvider } from '@/shell/breadcrumb_context';
import { PageBreadcrumbs } from '@/shell/page_breadcrumbs';
import { CommandPalette } from '@/shell/command_palette';
import { EulaGate } from '@/shell/eula_gate';
import { Toaster } from '@/components/ui/sonner';
import { TooltipProvider } from '@/components/ui/tooltip';
import { useSession } from '@/hooks/use_session';
import { hasAnyPortalAccess } from '@/lib/portal_access';
import { readSidebarCollapsed, persistSidebarCollapsed } from '@/lib/sidebar_transition';
import { TrackerHeaderProvider } from '@/lib/tracker_header_context';
import { listTrackerNavItems } from '@/lib/tracker_nav';
import { cn } from '@/lib/utils';

export function AppShell() {
  const { session, user } = useSession();
  const [signingOut, setSigningOut] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => readSidebarCollapsed());
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);

  const navItems = useMemo(() => {
    const items = listTrackerNavItems(user?.permissions);
    if (!hasAnyPortalAccess(user?.permissions)) {
      return items.filter((item) => item.path !== '/portals');
    }
    return items;
  }, [user?.permissions]);

  const handleSignOut = () => {
    setSigningOut(true);
    void logout()
      .catch(() => {
        // Session may already be cleared server-side; still leave the shell.
      })
      .finally(() => {
        window.location.replace('/login');
      });
  };

  const toggleSidebar = () => {
    setSidebarCollapsed((collapsed) => {
      const next = !collapsed;
      persistSidebarCollapsed(next);
      return next;
    });
  };

  return (
    <EulaGate>
      <TooltipProvider>
        <CommandPalette open={commandPaletteOpen} onOpenChange={setCommandPaletteOpen} />
        <Toaster />
        <a className="sr-only" href="#main-content">
          Skip to content
        </a>
        <div className="flex min-h-screen flex-col">
          <AdminDevBanner />
          <div className="flex min-h-0 flex-1">
            <aside
              className={cn(
                'flex w-48 shrink-0 flex-col gap-3 overflow-y-auto border-r border-zinc-200 bg-white p-3 dark:border-zinc-800 dark:bg-zinc-950',
                sidebarCollapsed && 'hidden',
              )}
            >
            <p>
              <strong>ad-event-processor</strong>
            </p>
            <nav aria-label="Main">
              {navItems.map((item) => (
                <NavLink
                  key={item.path}
                  end={item.path === '/dashboards/buyer'}
                  className="relative block rounded px-2 py-1.5 no-underline hover:bg-zinc-100 dark:hover:bg-zinc-800 [&[aria-current=page]]:bg-blue-50 [&[aria-current=page]]:font-semibold [&[aria-current=page]]:text-blue-700 dark:[&[aria-current=page]]:bg-blue-950/40 dark:[&[aria-current=page]]:text-blue-300"
                  to={item.path}
                >
                  {item.label}
                </NavLink>
              ))}
            </nav>
            {session ? (
              <p style={{ marginTop: 'auto' }}>
                <Button disabled={signingOut} loading={signingOut} type="button" onClick={handleSignOut}>
                  Sign out
                </Button>
              </p>
            ) : null}
          </aside>

          <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-zinc-50 dark:bg-zinc-950">
            <TrackerHeaderProvider>
              <BreadcrumbProvider>
                <header className="grid grid-cols-[minmax(0,1fr)_minmax(12rem,28rem)_minmax(0,1fr)] items-center gap-3 border-b border-zinc-200 bg-white px-3 py-2 dark:border-zinc-800 dark:bg-zinc-950">
                  <div className="flex min-w-0 items-center gap-2">
                    <TrackerShellSidebarToggle
                      collapsed={sidebarCollapsed}
                      onToggle={toggleSidebar}
                    />
                    <PageBreadcrumbs className="min-w-0 overflow-hidden" />
                  </div>
                  <div className="flex w-full max-w-md min-w-0 justify-center justify-self-center">
                    <TrackerShellHeaderSearch
                      onOpenCommandPalette={() => setCommandPaletteOpen(true)}
                    />
                  </div>
                  <div className="flex min-w-0 items-center justify-end justify-self-end gap-2">
                    <TrackerShellHeaderActions />
                  </div>
                </header>
                <main className="flex min-h-0 flex-1 flex-col overflow-hidden p-3" id="main-content" tabIndex={-1}>
                  <AppErrorBoundary layout="embedded">
                    <Outlet />
                  </AppErrorBoundary>
                </main>
              </BreadcrumbProvider>
            </TrackerHeaderProvider>
          </div>
        </div>
      </div>
      </TooltipProvider>
    </EulaGate>
  );
}
