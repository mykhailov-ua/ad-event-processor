import { resolveReportCatalogKey } from '@/lib/report_paths';

const CAMPAIGN_UUID_PATH =
  /^\/campaigns\/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})\/?$/i;

export function normalizeHref(href: string): string {
  if (href.startsWith('/')) {
    return href;
  }
  return `/${href}`;
}

export function resolveCommandPaletteHref(href: string): string {
  const normalized = normalizeHref(href.trim());
  if (!normalized) {
    return normalized;
  }

  const queryIndex = normalized.indexOf('?');
  const pathname = queryIndex >= 0 ? normalized.slice(0, queryIndex) : normalized;
  const search = queryIndex >= 0 ? normalized.slice(queryIndex + 1) : '';

  const campaignMatch = pathname.match(CAMPAIGN_UUID_PATH);
  if (campaignMatch) {
    const suffix = search ? `?${search}` : '';
    return `/campaigns/${campaignMatch[1]}/edit${suffix}`;
  }

  if (pathname.startsWith('/reports/')) {
    const rawKey = decodeURIComponent(pathname.slice('/reports/'.length)).replace(/^\/+|\/+$/g, '');
    if (rawKey && rawKey !== 'jobs') {
      const reportKey = resolveReportCatalogKey(rawKey);
      const params = new URLSearchParams(search);
      params.set('kind', 'report');
      params.set('report_key', reportKey);
      if (!params.has('entry')) {
        params.set('entry', `report-${reportKey}`);
      }
      return `/exports?${params.toString()}`;
    }
  }

  if (pathname === '/platform-campaigns') {
    return search
      ? `/integrations/platform-campaigns?${search}`
      : '/integrations/platform-campaigns';
  }

  return normalized;
}
