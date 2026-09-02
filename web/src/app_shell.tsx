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
        <div className="admin-app-frame">
          <AdminDevBanner />
          <div className={cn('admin-app', sidebarCollapsed && 'is-sidebar-collapsed')}>
            <aside className="admin-sidebar">
            <p>
              <strong>ad-event-processor</strong>
            </p>
            <nav aria-label="Main">
              {navItems.map((item) => (
                <NavLink
                  key={item.path}
                  end={item.path === '/dashboards/buyer'}
                  className="admin-nav-link"
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

          <div className="admin-main">
            <TrackerHeaderProvider>
              <BreadcrumbProvider>
                <header className="admin-header">
                  <div className="admin-header-start">
                    <TrackerShellSidebarToggle
                      collapsed={sidebarCollapsed}
                      onToggle={toggleSidebar}
                    />
                    <PageBreadcrumbs className="admin-header-breadcrumbs" />
                  </div>
                  <div className="admin-header-center">
                    <TrackerShellHeaderSearch
                      onOpenCommandPalette={() => setCommandPaletteOpen(true)}
                    />
                  </div>
                  <div className="admin-header-end">
                    <TrackerShellHeaderActions />
                  </div>
                </header>
                <main className="admin-content" id="main-content" tabIndex={-1}>
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
