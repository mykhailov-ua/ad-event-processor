import { NavLink } from 'react-router-dom';

import { Button } from '@/components/ui/button';
import type { TrackerNavItem } from '@/lib/tracker_nav';
import { cn } from '@/lib/utils';

export type AppSidebarProps = {
  collapsed: boolean;
  items: TrackerNavItem[];
  signingOut: boolean;
  onSignOut: () => void;
};

export function AppSidebar({ collapsed, items, signingOut, onSignOut }: AppSidebarProps) {
  return (
    <aside
      className={cn(
        'flex h-full min-h-0 w-52 shrink-0 flex-col overflow-hidden border-r border-border bg-card text-card-foreground',
        collapsed && 'hidden',
      )}
    >
      <div className="flex shrink-0 items-center gap-2 px-3 py-3">
        <span
          aria-hidden
          className="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-primary"
        >
          <span className="h-3 w-3 rounded-sm border-2 border-primary-foreground/80" />
        </span>
        <span className="truncate text-sm font-bold text-foreground">ad-event-processor</span>
      </div>

      <nav aria-label="Main" className="ui-scrollbar flex min-h-0 flex-1 flex-col gap-px overflow-y-auto px-2">
        {items.map((item) => {
          const Icon = item.icon;
          return (
            <NavLink
              key={item.path}
              end={item.path === '/dashboards/buyer'}
              className={({ isActive }) =>
                cn(
                  'admin-sidebar-nav-link',
                  isActive ? 'admin-sidebar-nav-link--active' : 'admin-sidebar-nav-link--idle',
                )
              }
              to={item.path}
            >
              <Icon aria-hidden className="h-4 w-4 shrink-0 opacity-90" />
              <span className="truncate">{item.label}</span>
            </NavLink>
          );
        })}
      </nav>

      <div className="shrink-0 border-t border-border p-2">
        <Button
          className="w-full"
          disabled={signingOut}
          loading={signingOut}
          type="button"
          variant="outline"
          onClick={onSignOut}
        >
          Sign out
        </Button>
      </div>
    </aside>
  );
}
