import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';

import {
  recordCommandPaletteOpen,
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
import { useCommandPaletteContextualState } from '@/shell/command_palette_contextual';

const SEARCH_DEBOUNCE_MS = 250;

// AbortError reject: useResource swallows; gated lane is not an operator error.
function skipLazyFetch(): Promise<never> {
  return Promise.reject(new DOMException('Skipped', 'AbortError'));
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
  const { actions: contextualActions, resolveRun } = useCommandPaletteContextualState();
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

  const routesResource = useResource(
    async (signal) => {
      if (!open) {
        return skipLazyFetch();
      }
      return fetchCommandPaletteRoutesCached(signal);
    },
    [open]
  );

  const recentsResource = useResource(
    async (signal) => {
      if (!open || !customerId) {
        return skipLazyFetch();
      }
      return fetchCommandPaletteRecentsCached(customerId, signal);
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

  const routes = routesResource.data ?? [];
  const recents = recentsResource.data ?? [];
  const contextualItems = useMemo(
    () => contextualActions.map((action) => action.item),
    [contextualActions]
  );
  const catalogItems = useMemo(() => {
    if (recents.length === 0) {
      return routes;
    }
    return routes.filter((route) => !recents.some((recent) => recent.id === route.id));
  }, [recents, routes]);

  const searchItems = searchResource.data?.items ?? [];
  const degraded = searchResource.data?.degraded === true;
  const catalogError = routesResource.error ?? recentsResource.error;
  const searchError = searchResource.error;
  const catalogLoading = routesResource.fetching || (customerId ? recentsResource.fetching : false);
  const searchLoading = searchResource.fetching;

  const catalogForbidden = catalogError instanceof ApiError && catalogError.status === 403;
  const searchForbidden = searchError instanceof ApiError && searchError.status === 403;
  const activeError = isSearching ? searchError : catalogError;
  const activeLoading = isSearching ? searchLoading : catalogLoading;
  const paletteForbidden = isSearching ? searchForbidden : catalogForbidden;

  const onSelectItem = useCallback(
    (item: CommandPaletteItem) => {
      const contextualRun = resolveRun(item.id);
      setOpen(false);
      if (contextualRun) {
        contextualRun();
        return;
      }
      if (customerId) {
        void recordCommandPaletteRecent({ customer_id: customerId, item }).catch(() => undefined);
      }
      navigate(resolveCommandPaletteHref(item.href));
    },
    [customerId, navigate, resolveRun, setOpen]
  );

  return {
    open,
    setOpen,
    query,
    setQuery,
    isSearching,
    contextualItems,
    catalogItems,
    recents,
    searchItems,
    degraded,
    catalogError,
    catalogLoading,
    searchError,
    searchLoading,
    activeError,
    activeLoading,
    paletteForbidden,
    onSelectItem,
  };
}
