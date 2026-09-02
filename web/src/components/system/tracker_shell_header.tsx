import { Bell, CircleUser, HelpCircle, Search } from 'lucide-react';
import { Link } from 'react-router-dom';

import { useTrackerHeaderSearch } from '@/lib/tracker_header_context';

export function TrackerShellHeaderSearch() {
  const search = useTrackerHeaderSearch();
  if (!search) {
    return <div aria-hidden className="h-9 w-full max-w-md" />;
  }

  return (
    <label className="relative w-full max-w-md" htmlFor="tracker-shell-search">
      <span className="sr-only">Search</span>
      <Search
        aria-hidden
        className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[#9aa0a8]"
      />
      <input
        className="tracker-shell-header-search"
        disabled={search.disabled}
        id="tracker-shell-search"
        placeholder={search.placeholder ?? 'Search'}
        value={search.value}
        onBlur={search.onApply}
        onChange={(event) => search.onChange(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === 'Enter') {
            event.preventDefault();
            search.onApply();
          }
        }}
      />
    </label>
  );
}

export function TrackerShellHeaderActions() {
  return (
    <div className="tracker-shell-header-actions">
      <Link aria-label="Account" className="tracker-shell-header-icon-btn" title="Account" to="/settings">
        <CircleUser className="h-4 w-4" />
      </Link>
      <Link aria-label="FAQ" className="tracker-shell-header-icon-btn" title="FAQ" to="/docs">
        <HelpCircle className="h-4 w-4" />
      </Link>
      <button
        aria-label="Notifications"
        className="tracker-shell-header-icon-btn"
        title="Notifications"
        type="button"
      >
        <Bell className="h-4 w-4" />
      </button>
    </div>
  );
}
