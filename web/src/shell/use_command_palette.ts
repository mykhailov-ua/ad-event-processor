import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';

import {
  listCommandPaletteRecents,
  recordCommandPaletteOpen,
  recordCommandPaletteRecent,
  searchCommandPalette,
  type CommandPaletteItem,
} from '@/api/command_palette_api';
import { ApiError } from '@/api/client';
import { useResource } from '@/api/use_resource';
import { fetchCommandPaletteRoutesCached } from '@/lib/command_palette_routes_cache';
import { useSession } from '@/hooks/use_session';

const SEARCH_DEBOUNCE_MS = 250;

function skipLazyFetch(): Promise<never> {
  return Promise.reject(new DOMException('Skipped', 'AbortError'));
}

function normalizeHref(href: string): string {
  if (href.startsWith('/')) {
    return href;
  }
  return `/${href}`;
}

export type UseCommandPaletteOptions = {
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
};

export function useCommandPalette({
  open: controlledOpen,
  onOpenChange,
}: UseCommandPaletteOptions = {}) {
  const navigate = useNavigate();
  const { session } = useSession();
  const customerId = session?.default_customer_id ?? '';

  const [internalOpen, setInternalOpen] = useState(false);
  const open = controlledOpen ?? internalOpen;

  const setOpen = useCallback(
    (next: boolean) => {
      onOpenChange?.(next);
      if (controlledOpen === undefined) {
        setInternalOpen(next);
      }
    },
    [controlledOpen, onOpenChange]
  );

  const [query, setQuery] = useState('');
  const trimmedQuery = query.trim();
  const isSearching = trimmedQuery.length >= 2;
  const [debouncedQuery, setDebouncedQuery] = useState('');

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        setOpen(true);
        setQuery('');
        setDebouncedQuery('');
        void recordCommandPaletteOpen({ source: 'keyboard' }).catch(() => undefined);
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [setOpen]);

  useEffect(() => {
    if (!open || !isSearching || !customerId) {
      setDebouncedQuery('');
      return undefined;
    }
    const timer = window.setTimeout(() => {
      setDebouncedQuery(trimmedQuery);
    }, SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(timer);
  }, [customerId, isSearching, open, trimmedQuery]);

  const catalogResource = useResource(
    async (signal) => {
      if (!open) {
        return skipLazyFetch();
      }
      const [routes, recentsResponse] = await Promise.all([
        fetchCommandPaletteRoutesCached(signal),
        customerId
          ? listCommandPaletteRecents(customerId, signal)
          : Promise.resolve({ items: [], total: 0 }),
      ]);
      return {
        routes,
        recents: recentsResponse.items ?? [],
      };
    },
    [customerId, open]
  );

  const searchResource = useResource(
    async (signal) => {
      if (!open || debouncedQuery.length < 2 || !customerId) {
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
    [customerId, debouncedQuery, open]
  );

  const routes = catalogResource.data?.routes ?? [];
  const recents = catalogResource.data?.recents ?? [];
  const catalogItems = useMemo(() => {
    if (recents.length === 0) {
      return routes;
    }
    return routes.filter((route) => !recents.some((recent) => recent.id === route.id));
  }, [recents, routes]);

  const searchItems = searchResource.data?.items ?? [];
  const degraded = searchResource.data?.degraded === true;
  const catalogError = catalogResource.error;
  const searchError = searchResource.error;
  const catalogLoading = catalogResource.fetching;
  const searchLoading = searchResource.fetching;

  const activeError = isSearching ? searchError : catalogError;
  const activeLoading = isSearching ? searchLoading : catalogLoading;
  const paletteForbidden = activeError instanceof ApiError && activeError.status === 403;

  const onSelectItem = useCallback(
    (item: CommandPaletteItem) => {
      setOpen(false);
      if (customerId) {
        void recordCommandPaletteRecent({ customer_id: customerId, item }).catch(() => undefined);
      }
      navigate(normalizeHref(item.href));
    },
    [customerId, navigate, setOpen]
  );

  return {
    open,
    setOpen,
    query,
    setQuery,
    isSearching,
    catalogItems,
    recents,
    searchItems,
    degraded,
    activeError,
    activeLoading,
    paletteForbidden,
    onSelectItem,
  };
}
