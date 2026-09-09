const EXPORT_HUB_RETURN_STORAGE_KEY = 'aed-export-hub-return-v1';

export type InAppLocation = {
  pathname: string;
  search: string;
  hash: string;
};

export function parseInAppLocation(href: string): InAppLocation {
  const normalized = href.trim();
  if (!normalized.startsWith('/') || normalized.startsWith('//')) {
    return { pathname: '/campaigns', search: '', hash: '' };
  }

  const hashIndex = normalized.indexOf('#');
  const hash = hashIndex >= 0 ? normalized.slice(hashIndex) : '';
  const withoutHash = hashIndex >= 0 ? normalized.slice(0, hashIndex) : normalized;
  const queryIndex = withoutHash.indexOf('?');

  if (queryIndex < 0) {
    return { pathname: withoutHash || '/', search: '', hash };
  }

  return {
    pathname: withoutHash.slice(0, queryIndex) || '/',
    search: withoutHash.slice(queryIndex),
    hash,
  };
}

export function rememberExportHubReturnPath(href: string): void {
  const target = parseInAppLocation(href);
  if (!target.pathname.startsWith('/campaigns')) {
    return;
  }
  try {
    sessionStorage.setItem(
      EXPORT_HUB_RETURN_STORAGE_KEY,
      `${target.pathname}${target.search}${target.hash}`
    );
  } catch {
    // private mode / quota
  }
}

export function readExportHubReturnPath(fallback = '/campaigns'): string {
  try {
    const raw = sessionStorage.getItem(EXPORT_HUB_RETURN_STORAGE_KEY)?.trim();
    if (raw && raw.startsWith('/') && !raw.startsWith('//')) {
      return raw;
    }
  } catch {
    // ignore
  }
  return fallback;
}
