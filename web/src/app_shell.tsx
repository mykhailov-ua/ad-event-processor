import { useMemo, useState, type ReactNode } from 'react';
import { NavLink, Outlet } from 'react-router-dom';
import {
  Activity,
  BookOpen,
  Layers3,
  LayoutGrid,
  LogOut,
  MoreHorizontal,
  Palette,
  PanelLeftClose,
  PanelLeftOpen,
  Search,
  type LucideIcon,
} from 'lucide-react';

import { logout } from '@/api/auth_api';

import { BreadcrumbProvider } from '@/components/system/breadcrumb_context';
import { PageBreadcrumbs } from '@/components/system/page_breadcrumbs';
import { CommandPalette } from '@/components/system/command_palette';
import { EulaGate } from '@/components/system/eula_gate';
import { ThemeToggle } from '@/components/system/theme_toggle';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Toaster } from '@/components/ui/sonner';
import { TooltipProvider } from '@/components/ui/tooltip';
import { useSession } from '@/hooks/use_session';
import { filterNavGroups, NAV_GROUPS, type NavGroup, type NavItem } from '@/lib/nav_config';
import { hasAnyPortalAccess } from '@/lib/portal_access';
import { persistSidebarCollapsed, readSidebarCollapsed } from '@/lib/sidebar_transition';
import { cn } from '@/lib/utils';

const collapsedSidebarPillClass =
  'flex h-9 w-9 shrink-0 items-center justify-center rounded-full p-0 text-xs font-medium leading-none whitespace-nowrap';

function navGroupIcon(id: string): LucideIcon {
  switch (id) {
    case 'core':
      return LayoutGrid;
    case 'operations':
      return Activity;
    case 'platform':
      return Layers3;
    case 'content':
      return Palette;
    case 'more':
      return MoreHorizontal;
    default:
      return LayoutGrid;
  }
}

function NavGroupLabel({ children, icon: Icon }: { children: ReactNode; icon: LucideIcon }) {
  return (
    <div className="flex items-center gap-2 px-2">
      <Icon aria-hidden="true" className="h-3.5 w-3.5 shrink-0 text-foreground/55" />
      <p className="text-xs font-semibold uppercase tracking-[0.14em] text-foreground/75">
        {children}
      </p>
    </div>
  );
}

function filterGroupsForPortals(groups: NavGroup[], permissions: string[] | undefined): NavGroup[] {
  return groups.map((group) => ({
    ...group,
    items: group.items.filter(
      (item) => item.path !== '/portals' || hasAnyPortalAccess(permissions),
    ),
  }));
}

function matchesNavQuery(item: NavItem, query: string): boolean {
  if (!query) {
    return true;
  }
  const haystack = `${item.label} ${item.path}`.toLowerCase();
  return haystack.includes(query);
}

function SidebarNavLink({
  abbrev,
  collapsed,
  end,
  label,
  to,
}: {
  abbrev?: string;
  collapsed: boolean;
  end?: boolean;
  label: string;
  to: string;
}) {
  const glyph = abbrev ?? label.charAt(0);

  const link = (
    <NavLink
      aria-label={collapsed ? label : undefined}
      end={end}
      title={collapsed ? label : undefined}
      to={to}
      className={({ isActive }) =>
        cn(
          'transition-colors',
          collapsed
            ? collapsedSidebarPillClass
            : 'block rounded-full px-3 py-2 whitespace-nowrap',
          isActive
            ? 'bg-secondary font-medium text-foreground'
            : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground',
        )
      }
    >
      {collapsed ? glyph : label}
    </NavLink>
  );

  if (!collapsed) {
    return link;
  }

  return <div className="flex justify-center">{link}</div>;
}

export function AppShell() {
  const { session, user } = useSession();
  const [navQuery, setNavQuery] = useState('');
  const [collapsed, setCollapsed] = useState(() => readSidebarCollapsed());
  const [signingOut, setSigningOut] = useState(false);

  const toggleCollapsed = () => {
    setCollapsed((value) => {
      const next = !value;
      persistSidebarCollapsed(next);
      return next;
    });
  };

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

  const navGroups = useMemo(() => {
    const filtered = filterNavGroups(NAV_GROUPS, user?.permissions);
    const withPortals = filterGroupsForPortals(filtered, user?.permissions);
    const query = navQuery.trim().toLowerCase();
    if (!query) {
      return withPortals;
    }
    return withPortals
      .map((group) => ({
        ...group,
        items: group.items.filter((item) => matchesNavQuery(item, query)),
      }))
      .filter((group) => group.items.length > 0);
  }, [navQuery, user?.permissions]);

  return (
    <EulaGate>
      <TooltipProvider>
        <CommandPalette />
        <Toaster />
        <a
          className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded-md focus:bg-primary focus:px-3 focus:py-2 focus:text-sm focus:text-primary-foreground"
          href="#main-content"
        >
          Skip to content
        </a>
        <div className="flex h-screen overflow-hidden bg-background">
          <aside
            className={cn(
              'flex h-full shrink-0 flex-col overflow-hidden bg-card/60',
              collapsed ? 'w-16' : 'w-64',
            )}
          >
            <div
              className={cn(
                'shrink-0',
                collapsed
                  ? 'grid w-full place-items-center gap-1 py-3'
                  : 'flex items-center justify-between gap-2 p-4 pb-2',
              )}
            >
              {!collapsed ? (
                <p className="min-w-0 flex-1 truncate text-sm font-semibold">ad-event-processor</p>
              ) : null}
              <div className={cn('flex shrink-0 items-center', collapsed ? 'flex-col gap-1' : 'gap-1')}>
                <ThemeToggle />
                <Button
                  aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
                  className={cn('shrink-0 p-0', collapsed ? 'size-9' : 'h-8 w-8')}
                  onClick={toggleCollapsed}
                  size={collapsed ? 'icon' : 'sm'}
                  type="button"
                  variant="ghost"
                >
                  {collapsed ? (
                    <PanelLeftOpen className="h-4 w-4" />
                  ) : (
                    <PanelLeftClose className="h-4 w-4" />
                  )}
                </Button>
              </div>
            </div>

            {!collapsed ? (
              <div className="shrink-0 px-4 pb-2">
                <div className="relative">
                  <Search className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    aria-label="Filter navigation"
                    className="h-9 pl-8 text-sm"
                    placeholder="Filter nav…"
                    value={navQuery}
                    onChange={(event) => setNavQuery(event.target.value)}
                  />
                </div>
              </div>
            ) : null}

            <nav
              aria-label="Main"
              className={cn(
                'ui-scrollbar min-h-0 flex-1 overflow-y-auto py-2 text-sm',
                collapsed ? 'overflow-x-hidden px-0' : 'px-2',
              )}
            >
              <div className={cn('flex flex-col', collapsed ? 'gap-1' : 'gap-2')}>
                {navGroups.map((group, index) => (
                  <div
                    key={group.id}
                    className={cn(
                      'flex flex-col gap-1',
                      !collapsed && index > 0 && 'border-t border-border/40 pt-3',
                    )}
                  >
                    {!collapsed ? (
                      <NavGroupLabel icon={navGroupIcon(group.id)}>{group.label}</NavGroupLabel>
                    ) : null}
                    {group.items.map((item) => (
                      <SidebarNavLink
                        key={item.path}
                        collapsed={collapsed}
                        label={item.label}
                        to={item.path}
                      />
                    ))}
                  </div>
                ))}
                <div
                  className={cn(
                    'flex flex-col gap-1',
                    !collapsed && 'border-t border-border/40 pt-3',
                  )}
                >
                  {!collapsed ? <NavGroupLabel icon={BookOpen}>Help</NavGroupLabel> : null}
                  <SidebarNavLink
                    collapsed={collapsed}
                    end={false}
                    label="Documentation"
                    to="/docs"
                  />
                </div>
              </div>
            </nav>

            <div
              className={cn(
                'shrink-0 border-t border-border/40 text-xs text-muted-foreground',
                collapsed
                  ? 'grid w-full place-items-center gap-2 py-3'
                  : 'flex flex-col gap-2 p-4',
              )}
            >
              {session ? (
                <>
                  {!collapsed ? (
                    <div className="flex w-full items-center justify-between gap-2">
                      <div className="min-w-0">
                        {session.role ? (
                          <p className="font-medium capitalize">{session.role}</p>
                        ) : null}
                        {session.timezone ? <p>{session.timezone}</p> : null}
                      </div>
                      <p className="shrink-0 text-muted-foreground/80">Search pages… ⌘K</p>
                    </div>
                  ) : (
                    <p className="text-center text-muted-foreground/80">⌘K</p>
                  )}
                  <Button
                    aria-label="Sign out"
                    className={cn(collapsed ? 'size-9 p-0' : 'w-full')}
                    disabled={signingOut}
                    onClick={handleSignOut}
                    size={collapsed ? 'icon' : 'sm'}
                    type="button"
                    variant="outline"
                  >
                    <LogOut className={cn('h-4 w-4', !collapsed && 'mr-2')} />
                    {!collapsed ? (signingOut ? 'Signing out…' : 'Sign out') : null}
                  </Button>
                </>
              ) : null}
            </div>
          </aside>

          <main
            className="ui-scrollbar min-h-0 min-w-0 flex-1 overflow-x-hidden overflow-y-auto"
            id="main-content"
            tabIndex={-1}
          >
            <BreadcrumbProvider>
              <div className="min-w-0 max-w-full p-6 lg:p-8">
                <PageBreadcrumbs />
                <Outlet />
              </div>
            </BreadcrumbProvider>
          </main>
        </div>
      </TooltipProvider>
    </EulaGate>
  );
}
