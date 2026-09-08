import { useEffect, useMemo, useState } from 'react';
import { Outlet } from 'react-router-dom';

import { logout } from '@/api/auth_api';
import { recordCommandPaletteOpen } from '@/api/command_palette_api';
import {
  TrackerShellHeaderActions,
  TrackerShellHeaderSearch,
  TrackerShellSidebarToggle,
} from '@/shell/tracker_shell_header';
import { PageCanvasInset } from '@/shell/page_layout';
import { shellChrome } from '@/shell/shell_chrome';
import { AppMobileNavSheet, AppSidebar } from '@/shell/app_sidebar';
import { AppErrorBoundary } from '@/shell/app_error_boundary';
import { RoutePermissionGuard } from '@/shell/permission_gate';
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
import { listTrackerNavGroups } from '@/lib/tracker_nav';
import { navigationExpanded } from '@/lib/navigation_toggle';

export function AppShell() {
  const { session, user } = useSession();
  const [signingOut, setSigningOut] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => readSidebarCollapsed());
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);
  const [viewportIsMdUp, setViewportIsMdUp] = useState(
    () => typeof window !== 'undefined' && window.matchMedia('(min-width: 768px)').matches
  );

  useEffect(() => {
    const media = window.matchMedia('(min-width: 768px)');
    const onViewportChange = () => {
      setViewportIsMdUp(media.matches);
      if (media.matches) {
        setMobileNavOpen(false);
      }
    };
    media.addEventListener('change', onViewportChange);
    return () => media.removeEventListener('change', onViewportChange);
  }, []);

  const navGroups = useMemo(() => {
    const groups = listTrackerNavGroups(user?.permissions);
    if (hasAnyPortalAccess(user?.permissions)) {
      return groups;
    }
    return groups
      .map((group) => ({
        ...group,
        items: group.items.filter((item) => item.path !== '/portals'),
      }))
      .filter((group) => group.items.length > 0);
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

  const handleNavToggle = () => {
    if (viewportIsMdUp) {
      toggleSidebar();
      return;
    }
    setMobileNavOpen((open) => !open);
  };

  const navigationExpandedState = navigationExpanded(
    viewportIsMdUp,
    sidebarCollapsed,
    mobileNavOpen
  );

  return (
    <EulaGate>
      <TooltipProvider>
        <CommandPalette open={commandPaletteOpen} onOpenChange={setCommandPaletteOpen} />
        <Toaster />
        <a className="sr-only" href="#main-content">
          Skip to content
        </a>
        <div className="flex h-dvh flex-col overflow-hidden">
          <div className="flex min-h-0 flex-1 overflow-hidden">
            {session ? (
              <>
                <AppSidebar
                  collapsed={sidebarCollapsed}
                  groups={navGroups}
                  signingOut={signingOut}
                  onSignOut={handleSignOut}
                />
                <AppMobileNavSheet
                  groups={navGroups}
                  open={mobileNavOpen}
                  signingOut={signingOut}
                  onOpenChange={setMobileNavOpen}
                  onSignOut={handleSignOut}
                />
              </>
            ) : null}

            <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-background">
              <TrackerHeaderProvider>
                <BreadcrumbProvider>
                  <header className={shellChrome.trackerHeaderClass}>
                    <div className="flex min-w-0 items-center gap-2">
                      <TrackerShellSidebarToggle
                        expanded={navigationExpandedState}
                        onToggle={handleNavToggle}
                      />
                      <PageBreadcrumbs className="min-w-0 overflow-x-auto" />
                    </div>
                    <div className="flex w-full max-w-md min-w-0 justify-center justify-self-center">
                      <TrackerShellHeaderSearch
                        onOpenCommandPalette={() => {
                          void recordCommandPaletteOpen({ source: 'ui' }).catch(() => undefined);
                          setCommandPaletteOpen(true);
                        }}
                      />
                    </div>
                    <div className="flex min-w-0 items-center justify-end justify-self-end gap-2">
                      <TrackerShellHeaderActions />
                    </div>
                  </header>
                  <main
                    className="flex min-h-0 flex-1 flex-col overflow-hidden bg-background"
                    id="main-content"
                    tabIndex={-1}
                  >
                    <div className="ui-scrollbar min-h-0 min-w-0 flex-1 overflow-x-hidden overflow-y-auto">
                      <PageCanvasInset>
                        <AppErrorBoundary layout="embedded">
                          <RoutePermissionGuard>
                            <Outlet />
                          </RoutePermissionGuard>
                        </AppErrorBoundary>
                      </PageCanvasInset>
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
