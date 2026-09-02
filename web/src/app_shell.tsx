import { useMemo, useState, type ComponentType } from 'react';
import { NavLink, Outlet, useLocation } from 'react-router-dom';
import { LogOut, Menu } from 'lucide-react';

import { logout } from '@/api/auth_api';
import { TrackerShellHeaderActions, TrackerShellHeaderSearch } from '@/components/system/tracker_shell_header';
import { BreadcrumbProvider } from '@/components/system/breadcrumb_context';
import { PageBreadcrumbs } from '@/components/system/page_breadcrumbs';
import { CommandPalette } from '@/components/system/command_palette';
import { EulaGate } from '@/components/system/eula_gate';
import { Toaster } from '@/components/ui/sonner';
import { TooltipProvider } from '@/components/ui/tooltip';
import { useSession } from '@/hooks/use_session';
import { hasAnyPortalAccess } from '@/lib/portal_access';
import { listTrackerNavItems } from '@/lib/tracker_nav';
import { TrackerHeaderProvider } from '@/lib/tracker_header_context';
import { cn } from '@/lib/utils';

function isTrackerWorkspacePath(pathname: string): boolean {
  return (
    pathname === '/campaigns' ||
    pathname.startsWith('/campaigns/') ||
    pathname === '/customers' ||
    pathname.startsWith('/customers/')
  );
}

function TrackerSidebarLink({
  end,
  icon: Icon,
  label,
  to,
}: {
  end?: boolean;
  icon: ComponentType<{ className?: string }>;
  label: string;
  to: string;
}) {
  return (
    <NavLink
      end={end}
      title={label}
      to={to}
      className={({ isActive }) =>
        cn(
          'tracker-shell-nav-link',
          isActive && 'tracker-shell-nav-link-active',
        )
      }
    >
      <Icon aria-hidden className="h-4 w-4 shrink-0 text-[#8a8a8a]" />
      <span className="truncate">{label}</span>
    </NavLink>
  );
}

export function AppShell() {
  const { session, user } = useSession();
  const location = useLocation();
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [signingOut, setSigningOut] = useState(false);

  const trackerWorkspace = isTrackerWorkspacePath(location.pathname);

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

  return (
    <EulaGate>
      <TooltipProvider>
        <TrackerHeaderProvider>
        <CommandPalette />
        <Toaster />
        <a
          className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded-md focus:bg-primary focus:px-3 focus:py-2 focus:text-sm focus:text-primary-foreground"
          href="#main-content"
        >
          Skip to content
        </a>
        <div className="tracker-shell flex h-screen overflow-hidden">
          <aside
            className={cn(
              'tracker-shell-sidebar flex shrink-0 flex-col border-r border-[#e0e0e0] bg-[#f4f4f4] transition-[width] duration-200',
              sidebarOpen ? 'w-[13.5rem]' : 'w-0 overflow-hidden border-r-0',
            )}
          >
            <nav aria-label="Main" className="ui-scrollbar min-h-0 flex-1 overflow-y-auto py-2">
              {navItems.map((item) => (
                <TrackerSidebarLink
                  key={item.path}
                  end={item.path === '/dashboards/buyer'}
                  icon={item.icon}
                  label={item.label}
                  to={item.path}
                />
              ))}
            </nav>
            {session ? (
              <div className="shrink-0 border-t border-[#e5e5e5] p-2">
                <button
                  className="tracker-shell-nav-link w-full border-0 bg-transparent text-left"
                  disabled={signingOut}
                  type="button"
                  onClick={handleSignOut}
                >
                  <LogOut aria-hidden className="h-4 w-4 shrink-0 text-[#8a8a8a]" />
                  <span>{signingOut ? 'Signing out…' : 'Sign out'}</span>
                </button>
              </div>
            ) : null}
          </aside>

          <div className="flex min-h-0 min-w-0 flex-1 flex-col">
            <header className="tracker-shell-header">
              <div className="tracker-shell-header-left">
                <button
                  aria-expanded={sidebarOpen}
                  aria-label={sidebarOpen ? 'Collapse navigation' : 'Expand navigation'}
                  className="tracker-shell-menu-btn"
                  type="button"
                  onClick={() => setSidebarOpen((value) => !value)}
                >
                  <Menu className="h-5 w-5" />
                </button>
                <span className="tracker-shell-brand">ad-event-processor</span>
              </div>

              {trackerWorkspace ? (
                <div className="tracker-shell-header-center">
                  <TrackerShellHeaderSearch />
                </div>
              ) : (
                <div aria-hidden className="tracker-shell-header-center" />
              )}

              {trackerWorkspace ? (
                <TrackerShellHeaderActions />
              ) : (
                <div aria-hidden className="tracker-shell-header-actions w-[6.5rem]" />
              )}
            </header>

            <main
              className={cn(
                'ui-scrollbar min-h-0 min-w-0 flex-1 overflow-x-hidden overflow-y-auto',
                trackerWorkspace ? 'bg-white' : 'bg-background',
              )}
              id="main-content"
              tabIndex={-1}
            >
              <BreadcrumbProvider>
                {trackerWorkspace ? (
                  <Outlet />
                ) : (
                  <div className="min-w-0 max-w-full p-6 lg:p-8">
                    <PageBreadcrumbs />
                    <Outlet />
                  </div>
                )}
              </BreadcrumbProvider>
            </main>
          </div>
        </div>
        </TrackerHeaderProvider>
      </TooltipProvider>
    </EulaGate>
  );
}
