import { useRef } from 'react';

import { adminSpacing } from '@/lib/admin_spacing';
import type { NavGroup } from '@/lib/nav_config';
import { HeaderAccountMenu } from '@/shell/header_account_menu';
import { HeaderNavMenu } from '@/shell/header_nav_menu';
import { HeaderSearch } from '@/shell/header_search';
import { ThemeToggle } from '@/shell/theme_toggle';
import { cn } from '@/lib/utils';

export type AppHeaderProps = {
  navGroups: NavGroup[];
  signingOut?: boolean;
  onOpenMobileNav: () => void;
  onSignOut: () => void;
};

export function AppHeader({
  navGroups,
  signingOut = false,
  onOpenMobileNav,
  onSignOut,
}: AppHeaderProps) {
  const searchInputRef = useRef<HTMLInputElement>(null);

  return (
    <header
      className="fixed inset-x-0 top-0 z-[10000] h-12 border-b border-border bg-background text-foreground"
      role="banner"
    >
      <div className={cn(adminSpacing.flex.headerBar, adminSpacing.inset.headerX)}>
        <div className={adminSpacing.flex.headerStart}>
          <HeaderNavMenu navGroups={navGroups} onOpenMobileNav={onOpenMobileNav} />
        </div>
        <div className={adminSpacing.flex.headerEnd}>
          <ThemeToggle />
          <HeaderAccountMenu signingOut={signingOut} onSignOut={onSignOut} />
        </div>
        <div className={adminSpacing.flex.headerSearchOverlay}>
          <div className="pointer-events-auto w-full max-w-xl min-w-[12rem]">
            <HeaderSearch inputRef={searchInputRef} />
          </div>
        </div>
      </div>
    </header>
  );
}
