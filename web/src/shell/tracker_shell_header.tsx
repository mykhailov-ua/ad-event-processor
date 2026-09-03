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
      className="inline-flex h-8 w-8 items-center justify-center rounded-md hover:bg-zinc-100 dark:hover:bg-zinc-800"
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
      <div className="w-full max-w-md">
        <Input
          aria-label="Search"
          disabled={pageSearch.disabled}
          placeholder={pageSearch.placeholder ?? 'Search'}
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
        className="flex h-8 w-full items-center justify-between gap-2 rounded-md border border-zinc-200 bg-white px-3 text-sm text-zinc-500 hover:bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-950 dark:hover:bg-zinc-900"
        type="button"
        onClick={onOpenCommandPalette}
      >
        <span className="truncate text-left">Search routes, campaigns, reports...</span>
        <kbd className="hidden rounded border border-zinc-200 px-1.5 py-0.5 text-[10px] font-medium text-zinc-500 sm:inline dark:border-zinc-700">Ctrl+K</kbd>
      </button>
    </div>
  );
}

export function TrackerShellHeaderActions() {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <ThemeToggle />
      <Button asChild type="button" variant="secondary">
        <Link to="/settings">Account</Link>
      </Button>
      <Button asChild type="button" variant="secondary">
        <Link to="/docs">Docs</Link>
      </Button>
    </div>
  );
}
