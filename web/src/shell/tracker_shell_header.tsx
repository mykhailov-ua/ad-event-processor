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
      aria-expanded={expanded}
      aria-label={expanded ? 'Hide navigation menu' : 'Show navigation menu'}
     
      type="button"
      variant="outline"
      onClick={onToggle}
    >
      <PanelLeft aria-hidden  />
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
      <div >
        <Input
          aria-label="Search"
         
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
    <div >
      <Button
       
        type="button"
        variant="outline"
        onClick={onOpenCommandPalette}
      >
        <span >Search routes, campaigns, reports...</span>
        <kbd >
          Ctrl+K
        </kbd>
      </Button>
    </div>
  );
}

export function TrackerShellHeaderActions() {
  return (
    <div >
      <Link
       
        to="/settings"
      >
        Account
      </Link>
      <ThemeToggle />
    </div>
  );
}
