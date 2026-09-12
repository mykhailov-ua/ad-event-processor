import { filterNavGroups, type NavGroup, type NavItem } from '@/lib/nav_config';

export const CONTROL_PLANE_NAV_ENABLED = true;

export const CONTROL_PLANE_CORE_NAV_PATHS = [
  '/customers',
  '/campaigns',
  '/team',
  '/settings',
] as const;

export const CONTROL_PLANE_OPERATIONS_NAV_PATHS = [
  '/ops',
  '/audit',
  '/exports',
  '/integrations',
] as const;

export function controlPlaneBareFrameEnabled(): boolean {
  return import.meta.env.ADMIN_UI_BARE === '1';
}

export function controlPlaneCampaignReportPath(_campaignId: string): string | null {
  return null;
}

export function controlPlaneNavGroupsForSidebar(): NavGroup[] {
  return [
    { id: 'core', label: 'Core', items: controlPlaneCoreNavItems() },
    { id: 'operations', label: 'Operations', items: controlPlaneOperationsNavItems() },
  ];
}

function controlPlaneCoreNavItems(): NavItem[] {
  return [
    { path: '/customers', label: 'Customers', permission: 'customers:read' },
    {
      path: '/campaigns',
      label: 'Campaigns',
      permissionAny: ['campaigns:read', 'campaigns:read:masked'],
    },
    { path: '/team', label: 'Team', permissionAny: ['team:read', 'campaigns:read'] },
    { path: '/settings', label: 'Settings', permission: 'settings:read' },
  ];
}

function controlPlaneOperationsNavItems(): NavItem[] {
  return [
    { path: '/exports', label: 'Exports', permission: 'campaigns:read' },
    { path: '/ops', label: 'Ops', permission: 'shards:read' },
    { path: '/audit', label: 'Audit', permission: 'audit:read' },
    { path: '/integrations', label: 'Integrations', permission: 'campaigns:read' },
  ];
}

export function filterControlPlaneNavGroups(
  permissions: string[] | undefined
): NavGroup[] {
  return filterNavGroups(controlPlaneNavGroupsForSidebar(), permissions);
}
