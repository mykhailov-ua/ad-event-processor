import { PanelLeft } from 'lucide-react';
import { Link } from 'react-router-dom';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { ThemeToggle } from '@/shell/theme_toggle';
import { useTrackerHeaderSearch } from '@/lib/tracker_header_context';

export type TrackerShellSidebarToggleProps = {
  collapsed: boolean;
  mobileNavOpen?: boolean;
  onToggle: () => void;
};

export function TrackerShellSidebarToggle({
  collapsed,
  mobileNavOpen = false,
  onToggle,
}: TrackerShellSidebarToggleProps) {
  const navOpen = mobileNavOpen || !collapsed;

  return (
    <Button
      aria-expanded={navOpen}
      aria-label={navOpen ? 'Hide navigation menu' : 'Show navigation menu'}
      className="inline-flex size-7 items-center justify-center rounded-md p-0 hover:bg-accent hover:text-accent-foreground"
      type="button"
      variant="secondary"
      onClick={onToggle}
    >
      <PanelLeft aria-hidden className="h-4 w-4" />
    </Button>
  );
}

export type TrackerShellHeaderSearchProps = {
  onOpenCommandPalette: () => void;
};

export function TrackerShellHeaderSearch({ onOpenCommandPalette }: TrackerShellHeaderSearchProps) {
  const pageSearch = useTrackerHeaderSearch();

  if (pageSearch) {
    return (
      <div className="w-60 max-w-full">
        <Input
          aria-label="Search"
          className="min-h-7 border-border bg-muted/50 text-xs placeholder:text-muted-foreground text-foreground"
          disabled={pageSearch.disabled}
          placeholder={pageSearch.placeholder ?? 'id, name, url'}
          value={pageSearch.value}
          onBlur={pageSearch.onApply}
          onChange={(event) => pageSearch.onChange(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter') {
              event.preventDefault();
              pageSearch.onApply();
            }
          }}
        />
      </div>
    );
  }

  return (
    <div className="w-full max-w-md">
      <Button
        className="w-full justify-between gap-2 px-3 font-normal text-muted-foreground"
        type="button"
        variant="outline"
        onClick={onOpenCommandPalette}
      >
        <span className="whitespace-nowrap text-left">Search routes, campaigns, reports...</span>
        <kbd className="hidden rounded border border-border px-1.5 py-0.5 text-ui-mini font-medium text-muted-foreground sm:inline">
          Ctrl+K
        </kbd>
      </Button>
    </div>
  );
}

export function TrackerShellHeaderActions() {
  return (
    <div className="flex flex-nowrap items-center gap-4">
      <Link
        className="inline-flex min-h-7 items-center text-[13px] font-medium text-muted-foreground transition-colors hover:text-foreground"
        to="/docs"
      >
        Docs
      </Link>
      <Link
        className="inline-flex min-h-7 items-center text-[13px] font-semibold text-foreground transition-colors hover:text-foreground"
        to="/settings"
      >
        Account
      </Link>
      <ThemeToggle />
    </div>
  );
}
