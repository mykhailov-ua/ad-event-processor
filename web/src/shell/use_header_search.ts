import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';

import {
  recordCommandPaletteRecent,
  searchCommandPalette,
  type CommandPaletteItem,
} from '@/api/command_palette_api';
import { ApiError } from '@/api/client';
import { useResource } from '@/api/use_resource';
import { resolveCommandPaletteHref } from '@/lib/command_palette_href';
import { fetchCommandPaletteRecentsCached } from '@/lib/command_palette_recents_cache';
import { fetchCommandPaletteRoutesCached } from '@/lib/command_palette_routes_cache';
import { useSession } from '@/hooks/use_session';

const SEARCH_DEBOUNCE_MS = 250;
const MIN_SERVER_QUERY_LENGTH = 2;

// AbortError reject: useResource swallows; gated lane is not an operator error.
function skipLazyFetch(): Promise<never> {
  return Promise.reject(new DOMException('Skipped', 'AbortError'));
}

function matchesQuery(item: CommandPaletteItem, query: string): boolean {
  const haystack = `${item.label} ${item.meta ?? ''} ${item.href}`.toLowerCase();
  return haystack.includes(query);
}

export function useHeaderSearch() {
  const navigate = useNavigate();
  const { session } = useSession();
  const customerId = session?.default_customer_id ?? '';

  const [query, setQuery] = useState('');
  const trimmedQuery = query.trim();
  const normalizedQuery = trimmedQuery.toLowerCase();
  const isSearching = normalizedQuery.length > 0;
  const useServerSearch = normalizedQuery.length >= MIN_SERVER_QUERY_LENGTH;
  const [debouncedQuery, setDebouncedQuery] = useState('');

  useEffect(() => {
    if (!useServerSearch) {
      setDebouncedQuery('');
      return undefined;
    }
    const timer = window.setTimeout(() => {
      setDebouncedQuery(trimmedQuery);
    }, SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(timer);
  }, [trimmedQuery, useServerSearch]);

  const routesResource = useResource(
    (signal) => fetchCommandPaletteRoutesCached(signal),
    []
  );

  const routesReady = routesResource.data !== undefined;

  const recentsResource = useResource(
    async (signal) => {
      if (!customerId || !routesReady) {
        return skipLazyFetch();
      }
      return fetchCommandPaletteRecentsCached(customerId, signal);
    },
    [customerId, routesReady]
  );

  const searchResource = useResource(
    async (signal) => {
      if (!useServerSearch || debouncedQuery.length < MIN_SERVER_QUERY_LENGTH || !customerId) {
        return skipLazyFetch();
      }
      const response = await searchCommandPalette(
        { customer_id: customerId, q: debouncedQuery },
        signal
      );
      return {
        items: response.items ?? [],
        degraded: response.degraded === true,
      };
    },
    [customerId, debouncedQuery, useServerSearch]
  );

  const routes = routesResource.data ?? [];
  const recents = recentsResource.data ?? [];

  const localItems = useMemo(() => {
    if (!isSearching) {
      return [];
    }
    const localMatches = routes.filter((item) => matchesQuery(item, normalizedQuery));
    const recentMatches = recents.filter((item) => matchesQuery(item, normalizedQuery));
    const seen = new Set<string>();
    const merged: CommandPaletteItem[] = [];
    for (const item of [...recentMatches, ...localMatches]) {
      if (seen.has(item.id)) {
        continue;
      }
      seen.add(item.id);
      merged.push(item);
    }
    return merged;
  }, [isSearching, normalizedQuery, recents, routes]);

  const serverItems = searchResource.data?.items ?? [];
  const degraded = searchResource.data?.degraded === true;

  const items = useMemo(() => {
    if (!isSearching) {
      return [];
    }
    if (!useServerSearch) {
      return localItems;
    }
    const seen = new Set(localItems.map((item) => item.id));
    const merged = [...localItems];
    for (const item of serverItems) {
      if (seen.has(item.id)) {
        continue;
      }
      seen.add(item.id);
      merged.push(item);
    }
    return merged;
  }, [isSearching, localItems, serverItems, useServerSearch]);

  const catalogError = routesResource.error ?? recentsResource.error;
  const catalogLoading =
    routesResource.fetching || (customerId && routesReady && recentsResource.fetching);
  const activeError = useServerSearch ? searchResource.error : catalogError;
  const activeLoading =
    catalogLoading || (useServerSearch && searchResource.fetching && items.length === 0);
  const paletteForbidden = activeError instanceof ApiError && activeError.status === 403;

  const onSelectItem = useCallback(
    (item: CommandPaletteItem) => {
      setQuery('');
      setDebouncedQuery('');
      if (customerId) {
        void recordCommandPaletteRecent({ customer_id: customerId, item }).catch(() => undefined);
      }
      navigate(resolveCommandPaletteHref(item.href));
    },
    [customerId, navigate]
  );

  return {
    query,
    setQuery,
    items,
    isSearching,
    degraded,
    activeError,
    activeLoading,
    paletteForbidden,
    onSelectItem,
  };
}
