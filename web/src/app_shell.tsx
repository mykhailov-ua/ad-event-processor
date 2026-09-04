import { useMemo, useState } from 'react';
import { Outlet } from 'react-router-dom';

import { logout } from '@/api/auth_api';
import {
  TrackerShellHeaderActions,
  TrackerShellHeaderSearch,
  TrackerShellSidebarToggle,
} from '@/shell/tracker_shell_header';
import { AppSidebar } from '@/shell/app_sidebar';
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
        <div className="flex h-dvh flex-col overflow-hidden">
          <AdminDevBanner />
          <div className="flex min-h-0 flex-1 overflow-hidden">
            {session ? (
              <AppSidebar
                collapsed={sidebarCollapsed}
                items={navItems}
                signingOut={signingOut}
                onSignOut={handleSignOut}
              />
            ) : null}

            <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-background">
              <TrackerHeaderProvider>
                <BreadcrumbProvider>
                  <header className="grid h-11 shrink-0 grid-cols-[minmax(0,1fr)_minmax(12rem,28rem)_minmax(0,1fr)] items-center gap-3 border-b border-border bg-card px-5 text-card-foreground">
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
                  <main
                    className="flex min-h-0 flex-1 flex-col overflow-hidden bg-background p-4"
                    id="main-content"
                    tabIndex={-1}
                  >
                    <div className="flex min-h-0 min-w-0 flex-1 flex-col">
                      <AppErrorBoundary layout="embedded">
                        <Outlet />
                      </AppErrorBoundary>
                    </div>
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
