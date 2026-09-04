import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';

import {
  listCommandPaletteRecents,
  recordCommandPaletteOpen,
  recordCommandPaletteRecent,
  searchCommandPalette,
  type CommandPaletteItem,
} from '@/api/command_palette_api';
import { fetchCommandPaletteRoutesCached } from '@/lib/command_palette_routes_cache';
import { ApiError } from '@/api/client';
import { ErrorBlock } from '@/shell/error_block';
import { CommandPaletteRow } from '@/shell/command_palette_row';
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandList,
  CommandSeparator,
} from '@/components/ui/command';
import { useSession } from '@/hooks/use_session';

const SEARCH_DEBOUNCE_MS = 250;

export type CommandPaletteProps = {
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
};

function normalizeHref(href: string): string {
  if (href.startsWith('/')) {
    return href;
  }
  return `/${href}`;
}

export function CommandPalette({ open: controlledOpen, onOpenChange }: CommandPaletteProps = {}) {
  const navigate = useNavigate();
  const { session } = useSession();
  const customerId = session?.default_customer_id ?? '';

  const [internalOpen, setInternalOpen] = useState(false);
  const open = controlledOpen ?? internalOpen;

  function setOpen(next: boolean) {
    onOpenChange?.(next);
    if (controlledOpen === undefined) {
      setInternalOpen(next);
    }
  }

  const [query, setQuery] = useState('');
  const [routes, setRoutes] = useState<CommandPaletteItem[]>([]);
  const [recents, setRecents] = useState<CommandPaletteItem[]>([]);
  const [catalogLoading, setCatalogLoading] = useState(false);
  const [catalogError, setCatalogError] = useState<Error | undefined>();

  const [searchItems, setSearchItems] = useState<CommandPaletteItem[]>([]);
  const [searchLoading, setSearchLoading] = useState(false);
  const [searchError, setSearchError] = useState<Error | undefined>();
  const [degraded, setDegraded] = useState(false);

  const trimmedQuery = query.trim();
  const isSearching = trimmedQuery.length >= 2;

  const catalogItems = useMemo(() => {
    if (recents.length === 0) {
      return routes;
    }
    return routes.filter((route) => !recents.some((recent) => recent.id === route.id));
  }, [recents, routes]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        onOpenChange?.(true);
        if (controlledOpen === undefined) {
          setInternalOpen(true);
        }
        setQuery('');
        setSearchItems([]);
        setSearchError(undefined);
        setDegraded(false);
        void recordCommandPaletteOpen({ source: 'keyboard' }).catch(() => undefined);
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [controlledOpen, onOpenChange]);

  useEffect(() => {
    if (!open) {
      return;
    }

    const ctrl = new AbortController();

    async function loadCatalog(signal: AbortSignal) {
      setCatalogLoading(true);
      setCatalogError(undefined);
      try {
        const [nextRoutes, recentsResponse] = await Promise.all([
          fetchCommandPaletteRoutesCached(signal),
          customerId
            ? listCommandPaletteRecents(customerId, signal)
            : Promise.resolve({ items: [], total: 0 }),
        ]);
        setRoutes(nextRoutes);
        setRecents(recentsResponse.items ?? []);
      } catch (err) {
        if (signal.aborted) {
          return;
        }
        setCatalogError(err instanceof Error ? err : new Error(String(err)));
      } finally {
        if (!signal.aborted) {
          setCatalogLoading(false);
        }
      }
    }

    void loadCatalog(ctrl.signal);
    return () => ctrl.abort();
  }, [customerId, open]);

  useEffect(() => {
    if (!open || !isSearching || !customerId) {
      setSearchItems([]);
      setSearchLoading(false);
      setSearchError(undefined);
      return;
    }

    const ctrl = new AbortController();
    const timer = window.setTimeout(() => {
      setSearchLoading(true);
      setSearchError(undefined);
      void searchCommandPalette({ customer_id: customerId, q: trimmedQuery }, ctrl.signal)
        .then((response) => {
          setSearchItems(response.items ?? []);
          setDegraded(response.degraded === true);
        })
        .catch((err: unknown) => {
          if (ctrl.signal.aborted) {
            return;
          }
          setSearchError(err instanceof Error ? err : new Error(String(err)));
        })
        .finally(() => {
          if (!ctrl.signal.aborted) {
            setSearchLoading(false);
          }
        });
    }, SEARCH_DEBOUNCE_MS);

    return () => {
      ctrl.abort();
      window.clearTimeout(timer);
    };
  }, [customerId, isSearching, open, trimmedQuery]);

  function onSelectItem(item: CommandPaletteItem) {
    setOpen(false);
    if (customerId) {
      void recordCommandPaletteRecent({ customer_id: customerId, item }).catch(() => undefined);
    }
    navigate(normalizeHref(item.href));
  }

  const activeError = isSearching ? searchError : catalogError;
  const activeLoading = isSearching ? searchLoading : catalogLoading;
  const paletteForbidden = activeError instanceof ApiError && activeError.status === 403;

  return (
    <CommandDialog onOpenChange={setOpen} open={open} shouldFilter={false}>
      <p className="sr-only" id="command-palette-description">
        Search admin routes and entities. Press Ctrl+K or Cmd+K to reopen.
      </p>
      <CommandInput
        aria-label="Command palette search"
        placeholder="Search routes, campaigns, reports..."
        value={query}
        onValueChange={setQuery}
      />
      {degraded ? (
        <p className="px-3 pb-1 text-xs text-muted-foreground">Search results may be incomplete.</p>
      ) : null}
      {paletteForbidden ? (
        <div className="px-3 pb-3">
          <ErrorBlock title="Command palette forbidden" message={activeError.message} />
        </div>
      ) : activeError && !paletteForbidden ? (
        <div className="px-3 pb-3">
          <ErrorBlock title="Command palette failed" message={activeError.message} />
        </div>
      ) : (
        <CommandList aria-label="Command palette results">
          <CommandEmpty>
            {activeLoading ? 'Loading...' : isSearching ? 'No matches.' : 'No entries.'}
          </CommandEmpty>
          {isSearching ? (
            <CommandGroup heading="Results">
              {searchItems.map((item) => (
                <CommandPaletteRow key={item.id} item={item} onSelect={onSelectItem} />
              ))}
            </CommandGroup>
          ) : (
            <>
              {recents.length > 0 ? (
                <CommandGroup heading="Recent">
                  {recents.map((item) => (
                    <CommandPaletteRow key={item.id} item={item} onSelect={onSelectItem} />
                  ))}
                </CommandGroup>
              ) : null}
              {recents.length > 0 && catalogItems.length > 0 ? <CommandSeparator /> : null}
              {catalogItems.length > 0 ? (
                <CommandGroup heading="Routes">
                  {catalogItems.map((item) => (
                    <CommandPaletteRow key={item.id} item={item} onSelect={onSelectItem} />
                  ))}
                </CommandGroup>
              ) : null}
            </>
          )}
        </CommandList>
      )}
    </CommandDialog>
  );
}
