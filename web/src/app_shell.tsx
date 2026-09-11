import { useMemo, useState } from 'react';
import { Outlet } from 'react-router-dom';

import { logout } from '@/api/auth_api';
import { getTeamOverview } from '@/api/team_api';
import { useResource } from '@/api/use_resource';
import { AppHeader } from '@/shell/app_header';
import { AppMobileNavSheet } from '@/shell/app_sidebar';
import { AppRouteErrorBoundary } from '@/shell/app_error_boundary';
import { CommandPalette } from '@/shell/command_palette';
import { CommandPaletteContextualProvider } from '@/shell/command_palette_contextual';
import { RoutePermissionGuard } from '@/shell/permission_gate';
import { BreadcrumbProvider } from '@/shell/breadcrumb_context';
import { EulaGate } from '@/shell/eula_gate';
import { Toaster } from '@/components/ui/sonner';
import { TooltipProvider } from '@/components/ui/tooltip';
import { useSession } from '@/hooks/use_session';
import { listTrackerNavGroups } from '@/lib/tracker_nav';
import { sessionHasPermission } from '@/lib/session_permissions';
import { PageCanvasInset } from '@/shell/page_layout';
import { TrackerHeaderProvider } from '@/lib/tracker_header_context';

export function AppShell() {
  const { user, session } = useSession();
  const [signingOut, setSigningOut] = useState(false);
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const teamBadgeCustomerId = session?.default_customer_id ?? '';
  const canLoadTeamNavBadge = sessionHasPermission(user?.permissions, 'team:write');
  const { data: teamOverviewForNav } = useResource(
    (signal) => {
      if (!canLoadTeamNavBadge || !teamBadgeCustomerId) {
        return Promise.resolve(undefined);
      }
      return getTeamOverview({ customer_id: teamBadgeCustomerId }, signal);
    },
    [canLoadTeamNavBadge, teamBadgeCustomerId]
  );

  const navGroups = useMemo(() => {
    const groups = listTrackerNavGroups(user?.permissions);
    const pending = teamOverviewForNav?.pending_approvals_count ?? 0;
    if (pending <= 0) {
      return groups;
    }
    return groups.map((group) => ({
      ...group,
      items: group.items.map((item) =>
        item.path === '/team' ? { ...item, badgeCount: pending } : item
      ),
    }));
  }, [teamOverviewForNav?.pending_approvals_count, user?.permissions]);

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

  const outlet = (
    <AppRouteErrorBoundary layout="embedded">
      <RoutePermissionGuard>
        <Outlet />
      </RoutePermissionGuard>
    </AppRouteErrorBoundary>
  );

  return (
    <EulaGate>
      <CommandPaletteContextualProvider>
        <TooltipProvider>
          <Toaster />
          <CommandPalette />
          <div className="min-h-screen bg-background text-foreground">
            <a
              className="sr-only focus:not-sr-only focus:absolute focus:left-2 focus:top-2 focus:z-[10001] focus:rounded-md focus:border focus:border-border focus:bg-background focus:px-3 focus:py-2"
              href="#main-content"
            >
              Skip to content
            </a>
            <TrackerHeaderProvider>
              <BreadcrumbProvider>
                <AppHeader
                  navGroups={navGroups}
                  signingOut={signingOut}
                  onOpenMobileNav={() => setMobileNavOpen(true)}
                  onSignOut={handleSignOut}
                />
              </BreadcrumbProvider>
              <AppMobileNavSheet
                groups={navGroups}
                open={mobileNavOpen}
                signingOut={signingOut}
                onOpenChange={setMobileNavOpen}
                onSignOut={handleSignOut}
              />
              <main className="pt-12" id="main-content" tabIndex={-1}>
                <PageCanvasInset>{outlet}</PageCanvasInset>
              </main>
            </TrackerHeaderProvider>
          </div>
        </TooltipProvider>
      </CommandPaletteContextualProvider>
    </EulaGate>
  );
}
