import { REPORT_CATALOG, reportHref } from '../models/report.js';
import type { NavLink } from './nav_config.js';

const OPERATOR_ONLY_KEYS = new Set([
  'spend-velocity',
  'discrepancy-buy-sell',
  'customer-portfolio',
  'campaign-overview',
]);

const AUDIT_KEYS = new Set(['ivt-by-source']);

/** In-page tab targets for Telegram analytics (not sidebar rows — P0.2). */
export const TELEGRAM_REPORT_PAGES = [
  { path: '/reports/telegram', label: 'Summary', icon: 'send' },
  { path: '/reports/telegram/funnel', label: 'Funnel', icon: 'git-branch' },
  { path: '/reports/telegram/bots', label: 'Bots', icon: 'bot' },
  { path: '/reports/telegram/premium', label: 'Premium', icon: 'gem' },
  { path: '/reports/telegram/fraud', label: 'Fraud', icon: 'shield-alert' },
] as const;

function reportLinkPerms(key: string): Pick<NavLink, 'perm' | 'altPerm'> {
  if (AUDIT_KEYS.has(key)) {
    return { perm: 'audit:read' };
  }
  if (OPERATOR_ONLY_KEYS.has(key)) {
    return { perm: 'customers:read' };
  }
  return { perm: 'campaigns:read', altPerm: 'campaigns:read:masked' };
}

const TELEGRAM_PERMS: Pick<NavLink, 'perm' | 'altPerm'> = {
  perm: 'campaigns:read',
  altPerm: 'campaigns:read:masked',
};

/**
 * Sidebar Reports section (P0.1 / P0.2): hub + single Telegram entry only.
 */
export function sidebarReportsNavLinks(): NavLink[] {
  return [
    {
      to: '/reports',
      label: 'Reports',
      icon: 'bar-chart',
      ...TELEGRAM_PERMS,
    },
    {
      to: '/reports/telegram',
      label: 'Telegram Mini Apps',
      icon: 'send',
      ...TELEGRAM_PERMS,
    },
  ];
}

/**
 * Cmd+K entries for individual live reports and Telegram sub-pages (not sidebar).
 */
export function reportCommandPaletteLinks(): NavLink[] {
  const links: NavLink[] = [];

  for (const card of REPORT_CATALOG) {
    if (!card.live || card.retired || card.key === 'telegram') continue;
    links.push({
      to: reportHref(card.key),
      label: card.title,
      icon: card.icon,
      ...reportLinkPerms(card.key),
    });
  }

  for (const page of TELEGRAM_REPORT_PAGES) {
    links.push({
      to: page.path,
      label: `Telegram · ${page.label}`,
      icon: page.icon,
      ...TELEGRAM_PERMS,
    });
  }

  return links;
}
