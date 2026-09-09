import { NavLink } from 'react-router-dom';

import { Button } from '@/components/ui/button';
import { Sheet, SheetContent, SheetTitle } from '@/components/ui/sheet';
import { adminKit, adminSpacing, adminTypography } from '@/lib/admin_kit';
import { productDisplayName } from '@/lib/product_display_name';
import { AdminMark } from '@/shell/admin_mark';
import type { TrackerNavGroup, TrackerNavItem } from '@/lib/tracker_nav';
import { cn } from '@/lib/utils';

/** Baseline sidebar width and nav link chrome (pgAdmin tree selection palette). */
const SIDEBAR_WIDTH_CLASS = 'w-64';
const SIDEBAR_NAV_LINK_BASE = cn(
  'flex items-center no-underline transition-colors',
  adminSpacing.gap.sm,
  adminSpacing.inset.navItemX,
  adminSpacing.inset.navItemY,
  adminTypography.label,
  adminKit.nestedRadius
);
const SIDEBAR_NAV_ACTIVE = 'bg-admin-selection font-semibold text-foreground';
const SIDEBAR_NAV_IDLE = 'text-foreground hover:bg-accent';
const SIDEBAR_NAV_SCROLL_CLASS = cn(
  'scrollbar-admin flex min-h-0 flex-1 flex-col overflow-y-auto overscroll-y-contain pb-2',
  adminSpacing.inset.sidebarX
);

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
    <div className={cn('flex shrink-0 items-center border-b border-border', adminSpacing.gap.sm, adminSpacing.inset.bandLg)}>
      <span
        aria-hidden
        className={cn(
          'inline-flex h-8 w-8 shrink-0 items-center justify-center bg-primary text-primary-foreground',
          adminKit.controlRadius
        )}
      >
        <AdminMark className="h-4 w-4" />
      </span>
      <span className={cn('whitespace-nowrap tracking-tight', adminTypography.sectionTitle)}>
        {productDisplayName}
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
      end={item.path === '/exports' || item.path === '/audit' || item.path === '/settings'}
      className={({ isActive }) =>
        cn(SIDEBAR_NAV_LINK_BASE, isActive ? SIDEBAR_NAV_ACTIVE : SIDEBAR_NAV_IDLE)
      }
      to={item.path}
      onClick={onNavigate}
    >
      <Icon aria-hidden className="h-4 w-4 shrink-0 opacity-80" />
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
      {groups.map((group) => (
        <section key={group.id} className="mb-2">
          <h2 className={cn(adminSpacing.inset.navGroupLabel, adminKit.labelCaps)}>
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
        <div className={adminSpacing.inset.bandCompact}>
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
