import { PanelLeft } from 'lucide-react';
import { Link } from 'react-router-dom';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { ThemeToggle } from '@/shell/theme_toggle';
import { useTrackerHeaderSearch } from '@/lib/tracker_header_context';

export type TrackerShellSidebarToggleProps = {
  collapsed: boolean;
  onToggle: () => void;
};

export function TrackerShellSidebarToggle({
  collapsed,
  onToggle,
}: TrackerShellSidebarToggleProps) {
  return (
    <Button
      aria-expanded={!collapsed}
      aria-label={collapsed ? 'Show navigation menu' : 'Hide navigation menu'}
      className="inline-flex h-8 w-8 items-center justify-center rounded-md hover:bg-accent hover:text-accent-foreground"
      size="icon"
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
          className="h-8 border-border bg-muted/50 text-xs placeholder:text-muted-foreground text-foreground"
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
      <button
        className="flex h-8 w-full items-center justify-between gap-2 rounded-md border border-border bg-background px-3 text-sm text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
        type="button"
        onClick={onOpenCommandPalette}
      >
        <span className="truncate text-left">Search routes, campaigns, reports...</span>
        <kbd className="hidden rounded border border-border px-1.5 py-0.5 text-admin-mini font-medium text-muted-foreground sm:inline">Ctrl+K</kbd>
      </button>
    </div>
  );
}

export function TrackerShellHeaderActions() {
  return (
    <div className="flex flex-nowrap items-center gap-4">
      <Button asChild className="h-auto border-0 bg-transparent p-0 text-[13px] font-medium text-muted-foreground shadow-none hover:bg-transparent hover:text-foreground" type="button" variant="ghost">
        <Link to="/docs">Docs</Link>
      </Button>
      <Button asChild className="h-auto border-0 bg-transparent p-0 text-[13px] font-semibold text-foreground shadow-none hover:bg-transparent" type="button" variant="ghost">
        <Link to="/settings">Account</Link>
      </Button>
      <ThemeToggle />
    </div>
  );
}
