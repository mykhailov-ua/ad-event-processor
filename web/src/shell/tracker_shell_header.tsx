import { PanelLeft } from 'lucide-react';
import { Link } from 'react-router-dom';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { ThemeToggle } from '@/shell/theme_toggle';
import { useTrackerHeaderSearch } from '@/lib/tracker_header_context';

export type TrackerShellSidebarToggleProps = {
  expanded: boolean;
  onToggle: () => void;
};

export function TrackerShellSidebarToggle({ expanded, onToggle }: TrackerShellSidebarToggleProps) {
  return (
    <Button
      className="inline-flex size-7 items-center justify-center rounded-none p-0" aria-expanded={expanded}
      aria-label={expanded ? 'Hide navigation menu' : 'Show navigation menu'}
      type="button"
      variant="outline"
      onClick={onToggle}
    >
      <PanelLeft className="h-4 w-4" aria-hidden  />
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
      <div className="w-60 max-w-full" >
        <Input
          className="min-h-7 border-border bg-muted/50 text-xs placeholder:text-muted-foreground text-foreground" aria-label="Search"
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
    <div className="w-full max-w-md" >
      <Button
        className="grid w-full grid-cols-[1fr_auto] gap-2 px-3 font-normal text-muted-foreground" type="button"
        variant="outline"
        onClick={onOpenCommandPalette}
      >
        className="grid w-full grid-cols-[1fr_auto] gap-2 px-3 font-normal text-muted-foreground"
        <span className="whitespace-nowrap text-left">Search routes, campaigns, reports...</span>
        <span>Search routes, campaigns, reports...</span>
        <kbd>
          Ctrl+K
        </kbd>
      </Button>
    </div>
  );
}

export function TrackerShellHeaderActions() {
  return (
    <div>
      <Link
       
        className="inline-flex min-h-7 items-center text-[13px] font-semibold text-foreground transition-colors hover:text-foreground" to="/settings"
      >
        Account
      </Link>
      <ThemeToggle />
    </div>
  );
}
