import { useRef } from 'react';

import type { NavGroup } from '@/lib/nav_config';
import { HeaderAccountMenu } from '@/shell/header_account_menu';
import { HeaderNavMenu } from '@/shell/header_nav_menu';
import { HeaderSearch } from '@/shell/header_search';
import { ThemeToggle } from '@/shell/theme_toggle';

export type AppHeaderProps = {
  navGroups: NavGroup[];
  signingOut?: boolean;
  onSignOut: () => void;
};

export function AppHeader({ navGroups, signingOut = false, onSignOut }: AppHeaderProps) {
  const searchInputRef = useRef<HTMLInputElement>(null);

  return (
    <header
      className="fixed inset-x-0 top-0 z-[10000] h-12 border-b border-border bg-background text-foreground"
      role="banner"
    >
      <div className="relative flex h-full items-center justify-between gap-3 px-3 md:px-4">
        <div className="relative z-[1] flex min-w-0 items-center">
          <HeaderNavMenu navGroups={navGroups} />
        </div>
        <div className="relative z-[1] flex min-w-0 items-center justify-end gap-2">
          <ThemeToggle />
          <HeaderAccountMenu signingOut={signingOut} onSignOut={onSignOut} />
        </div>
        <div
          className="pointer-events-none absolute inset-x-0 top-0 flex h-12 items-center justify-center px-14 md:px-20"
        >
          <div className="pointer-events-auto w-full max-w-xl min-w-[12rem]">
            <HeaderSearch inputRef={searchInputRef} />
          </div>
        </div>
      </div>
    </header>
  );
}
