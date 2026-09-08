import { NavLink } from 'react-router-dom';

import { Button } from '@/components/ui/button';
import { Sheet, SheetContent, SheetTitle } from '@/components/ui/sheet';
import { adminKit } from '@/lib/admin_kit';
import { AdminMark } from '@/shell/admin_mark';
import type { TrackerNavGroup, TrackerNavItem } from '@/lib/tracker_nav';
import { cn } from '@/lib/utils';

/** Baseline sidebar width and nav link chrome (standard Tailwind scale; no fractional px). */
const SIDEBAR_WIDTH_CLASS = 'w-64';
const SIDEBAR_NAV_LINK_BASE =
  'flex items-center gap-2 rounded-none px-2.5 py-1 text-base leading-5 font-semibold no-underline transition-colors';
const SIDEBAR_NAV_ACTIVE =
  'border-l-2 border-primary bg-primary/10 pl-[calc(0.625rem-2px)] font-bold text-primary';
const SIDEBAR_NAV_IDLE = 'text-muted-foreground hover:bg-primary/5 hover:text-primary';
const SIDEBAR_NAV_SCROLL_CLASS =
  'ui-scrollbar flex min-h-0 flex-1 flex-col overflow-y-auto overscroll-y-contain px-2.5 pb-2';

export type AppSidebarProps = {
  collapsed: boolean;
  groups: TrackerNavGroup[];
  signingOut: boolean;
  onSignOut: () => void;
};

export type AppSidebarNavProps = {
  groups: TrackerNavGroup[];
  onNavigate?: () => void;
};

export type AppMobileNavSheetProps = {
  open: boolean;
  groups: TrackerNavGroup[];
  signingOut: boolean;
  onOpenChange: (open: boolean) => void;
  onSignOut: () => void;
};

function AppSidebarBrand() {
  return (
    <div className="flex shrink-0 items-center gap-2.5 px-4 py-4">
      <span
        aria-hidden
        className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-none bg-primary text-primary-foreground"
      >
        <AdminMark className="h-5 w-5" />
      </span>
      <span className="whitespace-nowrap text-sm font-bold leading-5 tracking-tight text-foreground">
        ad-event-processor
      </span>
    </div>
  );
}

function AppSidebarNavLink({
  item,
  onNavigate,
}: {
  item: TrackerNavItem;
  onNavigate?: () => void;
}) {
  const Icon = item.icon;
  return (
    <NavLink
      end={item.path === '/dashboards/buyer'}
      className={({ isActive }) =>
        cn(SIDEBAR_NAV_LINK_BASE, isActive ? SIDEBAR_NAV_ACTIVE : SIDEBAR_NAV_IDLE)
      }
      to={item.path}
      onClick={onNavigate}
    >
      <Icon aria-hidden className="h-5 w-5 shrink-0 opacity-90" />
      <span className="whitespace-nowrap">{item.label}</span>
    </NavLink>
  );
}

export function AppSidebarNav({ groups, onNavigate }: AppSidebarNavProps) {
  return (
    <nav
      aria-label="Main"
      className={SIDEBAR_NAV_SCROLL_CLASS}
      onWheel={(event) => event.stopPropagation()}
    >
      {groups.map((group, groupIndex) => (
        <section key={group.id} className="flex flex-col gap-px">
          <h2 className={cn(adminKit.labelCaps, 'px-2.5 pb-1', groupIndex === 0 ? 'pt-1' : 'pt-3')}>
            {group.label}
          </h2>
          {group.items.map((item) => (
            <AppSidebarNavLink key={item.path} item={item} onNavigate={onNavigate} />
          ))}
        </section>
      ))}
    </nav>
  );
}

export function AppMobileNavSheet({
  open,
  groups,
  signingOut,
  onOpenChange,
  onSignOut,
}: AppMobileNavSheetProps) {
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        className={cn(
          'flex min-h-0 max-w-[85vw] flex-col gap-0 overflow-hidden p-0',
          SIDEBAR_WIDTH_CLASS
        )}
        side="left"
      >
        <SheetTitle className="sr-only">Navigation</SheetTitle>
        <AppSidebarBrand />
        <AppSidebarNav groups={groups} onNavigate={() => onOpenChange(false)} />
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

export function AppSidebar({ collapsed, groups, signingOut, onSignOut }: AppSidebarProps) {
  return (
    <aside
      className={cn(
        'h-full min-h-0 shrink-0 flex-col overflow-hidden border-r border-border bg-card text-card-foreground',
        SIDEBAR_WIDTH_CLASS,
        collapsed ? 'hidden' : 'hidden md:flex'
      )}
    >
      <AppSidebarBrand />
      <AppSidebarNav groups={groups} />
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
