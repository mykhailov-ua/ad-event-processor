import { useMemo, useState } from 'react';
import { Outlet } from 'react-router-dom';

import { logout } from '@/api/auth_api';
import { AppHeader } from '@/shell/app_header';
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
import { PageCanvasInset } from '@/shell/page_layout';
import { TrackerHeaderProvider } from '@/lib/tracker_header_context';

export function AppShell() {
  const { user } = useSession();
  const [signingOut, setSigningOut] = useState(false);

  const navGroups = useMemo(() => listTrackerNavGroups(user?.permissions), [user?.permissions]);

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
                  onSignOut={handleSignOut}
                />
              </BreadcrumbProvider>
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
