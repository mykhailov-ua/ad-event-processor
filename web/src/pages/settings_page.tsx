import { Outlet } from 'react-router-dom';

import { useSession } from '@/hooks/use_session';
import type { SectionNavItem } from '@/lib/nav_config';
import { sessionHasPermission } from '@/lib/session_permissions';
import { PageChrome } from '@/shell/page_chrome';
import { SectionNav } from '@/shell/section_nav';

const SETTINGS_NAV_ITEMS: Array<SectionNavItem & { permission: string }> = [
  { path: '/settings', label: 'License', exact: true, permission: 'settings:read' },
  { path: '/settings/access', label: 'Access', exact: true, permission: 'access:read' },
];

export function SettingsPage() {
  const { user } = useSession();
  const navItems: SectionNavItem[] = SETTINGS_NAV_ITEMS.filter((item) =>
    sessionHasPermission(user?.permissions, item.permission)
  );

  return (
    <PageChrome title="Settings">
      <SectionNav items={navItems} label="Settings sections" variant="admin" />
      <div className="pt-6">
        <Outlet />
      </div>
    </PageChrome>
  );
}
