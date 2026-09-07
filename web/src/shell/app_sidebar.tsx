import { NavLink } from 'react-router-dom';

import { Button } from '@/components/ui/button';
import { Sheet, SheetContent, SheetTitle } from '@/components/ui/sheet';
import type { TrackerNavItem } from '@/lib/tracker_nav';
import { cn } from '@/lib/utils';

export type AppSidebarProps = {
  collapsed: boolean;
  items: TrackerNavItem[];
  signingOut: boolean;
  onSignOut: () => void;
};

export type AppSidebarNavProps = {
  items: TrackerNavItem[];
  onNavigate?: () => void;
};

export type AppMobileNavSheetProps = {
  open: boolean;
  items: TrackerNavItem[];
  signingOut: boolean;
  onOpenChange: (open: boolean) => void;
  onSignOut: () => void;
};

function AppSidebarBrand() {
  return (
    <div className="flex shrink-0 items-center gap-2 px-3 py-3">
      <span
        aria-hidden
        className="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-primary text-primary-foreground"
      >
        <span className="h-3 w-3 rounded-sm border-2 border-primary-foreground/90" />
      </span>
      <span className="whitespace-nowrap text-sm font-bold text-foreground">
        ad-event-processor
      </span>
    </div>
  );
}

export function AppSidebarNav({ items, onNavigate }: AppSidebarNavProps) {
  return (
    <nav
      aria-label="Main"
      className="ui-scrollbar flex min-h-0 flex-1 flex-col gap-px overflow-y-auto px-2"
    >
      {items.map((item) => {
        const Icon = item.icon;
        return (
          <NavLink
            key={item.path}
            end={item.path === '/dashboards/buyer'}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-1.5 rounded px-2 py-1 text-[13px] font-medium no-underline transition-colors',
                isActive
                  ? 'border-l-2 border-primary bg-primary/10 pl-[calc(0.5rem-2px)] font-semibold text-primary'
                  : 'text-muted-foreground hover:bg-primary/5 hover:text-primary'
              )
            }
            to={item.path}
            onClick={onNavigate}
          >
            <Icon aria-hidden className="h-4 w-4 shrink-0 opacity-90" />
            <span className="whitespace-nowrap">{item.label}</span>
          </NavLink>
        );
      })}
    </nav>
  );
}

export function AppMobileNavSheet({
  open,
  items,
  signingOut,
  onOpenChange,
  onSignOut,
}: AppMobileNavSheetProps) {
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="flex w-52 max-w-[85vw] flex-col gap-0 p-0 sm:max-w-xs" side="left">
        <SheetTitle className="sr-only">Navigation</SheetTitle>
        <AppSidebarBrand />
        <AppSidebarNav items={items} onNavigate={() => onOpenChange(false)} />
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
      </SheetContent>
    </Sheet>
  );
}

export function AppSidebar({ collapsed, items, signingOut, onSignOut }: AppSidebarProps) {
  return (
    <aside
      className={cn(
        'h-full min-h-0 w-52 shrink-0 flex-col overflow-hidden border-r border-border bg-card text-card-foreground',
        collapsed ? 'hidden' : 'hidden md:flex'
      )}
    >
      <AppSidebarBrand />
      <AppSidebarNav items={items} />
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
