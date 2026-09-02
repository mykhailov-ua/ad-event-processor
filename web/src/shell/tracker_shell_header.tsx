import { PanelLeft } from 'lucide-react';
import { Link } from 'react-router-dom';

import { Button } from '@/components/ui/button';
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
      className="admin-icon-btn"
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
      <div className="admin-header-search">
        <input
          aria-label="Search"
          className="admin-input admin-header-search-input"
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
    <div className="admin-header-search">
      <button
        className="admin-header-search-trigger"
        type="button"
        onClick={onOpenCommandPalette}
      >
        <span className="admin-header-search-placeholder">Search routes, campaigns, reports...</span>
        <kbd className="admin-header-search-kbd">Ctrl+K</kbd>
      </button>
    </div>
  );
}

export function TrackerShellHeaderActions() {
  return (
    <div className="admin-header-actions">
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
