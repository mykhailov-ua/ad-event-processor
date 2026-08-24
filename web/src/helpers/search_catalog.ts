import { navigate } from './spa_navigate.js';
import * as auth from './auth.js';
import * as storage from './storage.js';
import { NAV_GROUPS, NAV_OVERFLOW_LINKS, navLinkVisible, type NavLink } from './nav_config.js';
import { reportCommandPaletteLinks } from './nav_reports.js';
import { isCustomerUuid, shortCustomerId } from './customer_context.js';

function permLink(perm: string, altPerm?: string): NavLink {
  return { to: '', label: '', perm, altPerm };
}

export type SearchItem = {
  id: string;
  label: string;
  hint?: string;
  run: () => void;
};

export function buildSearchItems(onClose: () => void = () => {}): SearchItem[] {
  const user = auth.getUser();
  const permissions = user?.permissions ?? [];
  const items: SearchItem[] = [];

  const go = (path: string): void => {
    onClose();
    navigate(path);
  };

  for (const group of NAV_GROUPS) {
    for (const link of group.links) {
      if (!navLinkVisible(permissions, link)) continue;
      items.push({
        id: `nav-${link.to}`,
        label: link.label,
        hint: group.title,
        run: () => go(link.to),
      });
    }
  }

  for (const link of NAV_OVERFLOW_LINKS) {
    if (!navLinkVisible(permissions, link)) continue;
    items.push({
      id: `nav-overflow-${link.to}`,
      label: link.label,
      hint: 'More',
      run: () => go(link.to),
    });
  }

  for (const link of reportCommandPaletteLinks()) {
    if (!navLinkVisible(permissions, link)) continue;
    items.push({
      id: `report-${link.to}`,
      label: link.label,
      hint: 'Report',
      run: () => go(link.to),
    });
  }

  for (const id of storage.getRecentCustomerIds()) {
    if (navLinkVisible(permissions, permLink('customers:read'))) {
      items.push({
        id: `customer-${id}`,
        label: shortCustomerId(id, 12),
        hint: 'Recent customer',
        run: () => go(`/customers/${id}`),
      });
    }
    if (navLinkVisible(permissions, permLink('customers:read', 'billing:read'))) {
      items.push({
        id: `billing-${id}`,
        label: `Billing , ${shortCustomerId(id, 12)}`,
        hint: 'Customer billing',
        run: () => go(`/billing?customer_id=${encodeURIComponent(id)}`),
      });
    }
    if (navLinkVisible(permissions, permLink('campaigns:read', 'campaigns:read:masked'))) {
      items.push({
        id: `campaigns-${id}`,
        label: `Campaigns , ${shortCustomerId(id, 12)}`,
        hint: 'Customer campaigns',
        run: () => go(`/campaigns?customer_id=${encodeURIComponent(id)}`),
      });
    }
  }

  if (navLinkVisible(permissions, permLink('shards:read'))) {
    items.push({
      id: 'ops-outbox',
      label: 'Open ops outbox',
      hint: 'Operations',
      run: () => go('/ops'),
    });
  }
  if (navLinkVisible(permissions, permLink('campaigns:write'))) {
    items.push({
      id: 'action-pause-hint',
      label: 'Pause campaign (open detail first)',
      hint: 'Action',
      run: () => go('/campaigns'),
    });
  }

  return items;
}

export function filterSearchItems(
  items: SearchItem[],
  query: string,
  onClose: () => void = () => {},
  limit = 12
): SearchItem[] {
  const q = query.trim().toLowerCase();
  if (!q) return items.slice(0, limit);
  const matches = items.filter(
    (item) =>
      item.label.toLowerCase().includes(q) ||
      item.hint?.toLowerCase().includes(q) ||
      item.id.toLowerCase().includes(q)
  );
  if (isCustomerUuid(q)) {
    const id = query.trim();
    matches.unshift({
      id: `goto-customer-${id}`,
      label: `Open customer ${shortCustomerId(id, 12)}`,
      hint: 'UUID match',
      run: () => {
        onClose();
        navigate(`/customers/${id}`);
      },
    });
  }
  return matches.slice(0, limit);
}
