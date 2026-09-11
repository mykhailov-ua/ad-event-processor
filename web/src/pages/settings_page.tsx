import { Outlet } from 'react-router-dom';

import type { SectionNavItem } from '@/lib/nav_config';
import { PageChrome } from '@/shell/page_chrome';
import { SectionNav } from '@/shell/section_nav';

const SETTINGS_NAV_ITEMS: SectionNavItem[] = [
  { path: '/settings', label: 'License', exact: true },
  { path: '/settings/access', label: 'Access', exact: true },
];

export function SettingsPage() {
  return (
    <PageChrome title="Settings">
      <SectionNav items={SETTINGS_NAV_ITEMS} label="Settings sections" variant="admin" />
      <div className="pt-6">
        <Outlet />
      </div>
    </PageChrome>
  );
}
