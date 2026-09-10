import { NavLink, Outlet } from 'react-router-dom';

import { PageChrome } from '@/shell/page_chrome';
import { adminSpacing } from '@/lib/admin_spacing';
import { cn } from '@/lib/utils';

const SETTINGS_TABS = [
  { path: '/settings', label: 'License', end: true },
  { path: '/settings/access', label: 'Access', end: true },
] as const;

export function SettingsPage() {
  return (
    <PageChrome title="Settings">
      <nav className={cn('flex flex-wrap gap-4 border-b pb-3', adminSpacing.gap.md)}>
        {SETTINGS_TABS.map((tab) => (
          <NavLink
            className={({ isActive }) =>
              cn('text-sm font-medium', isActive ? 'text-foreground' : 'text-muted-foreground')
            }
            end={tab.end}
            key={tab.path}
            to={tab.path}
          >
            {tab.label}
          </NavLink>
        ))}
      </nav>
      <div className="pt-6">
        <Outlet />
      </div>
    </PageChrome>
  );
}
