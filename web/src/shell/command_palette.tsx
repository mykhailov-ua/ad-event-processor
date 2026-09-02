import { useCallback, useEffect, useMemo, useState } from 'react';
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
import { Badge } from '@/components/ui/badge';
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
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
  const setOpen = useCallback(
    (next: boolean) => {
      onOpenChange?.(next);
      if (controlledOpen === undefined) {
        setInternalOpen(next);
      }
    },
    [controlledOpen, onOpenChange],
  );
  const [query, setQuery] = useState('');
  const [routes, setRoutes] = useState<CommandPaletteItem[]>([]);
  const [recents, setRecents] = useState<CommandPaletteItem[]>([]);
  const [searchItems, setSearchItems] = useState<CommandPaletteItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | undefined>(undefined);
  const [degraded, setDegraded] = useState(false);

  const trimmedQuery = query.trim();
  const isSearching = trimmedQuery.length >= 2;

  const catalogItems = useMemo(() => {
    if (recents.length === 0) {
      return routes;
    }
    return routes.filter((route) => !recents.some((recent) => recent.id === route.id));
  }, [recents, routes]);

  const loadCatalog = useCallback(async (signal: AbortSignal) => {
    setLoading(true);
    setError(undefined);
    try {
      const [routes, recentsResponse] = await Promise.all([
        fetchCommandPaletteRoutesCached(signal),
        customerId
          ? listCommandPaletteRecents(customerId, signal)
          : Promise.resolve({ items: [], total: 0 }),
      ]);
      setRoutes(routes);
      setRecents(recentsResponse.items ?? []);
    } catch (err) {
      if (signal.aborted) {
        return;
      }
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      if (!signal.aborted) {
        setLoading(false);
      }
    }
  }, [customerId]);

  const openPalette = useCallback(() => {
    setOpen(true);
    setQuery('');
    setSearchItems([]);
    setDegraded(false);
    void recordCommandPaletteOpen({ source: 'keyboard' }).catch(() => undefined);
  }, [setOpen]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        openPalette();
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [openPalette]);

  useEffect(() => {
    if (!open) {
      return;
    }
    const ctrl = new AbortController();
    void loadCatalog(ctrl.signal);
    return () => ctrl.abort();
  }, [loadCatalog, open]);

  useEffect(() => {
    if (!open || !isSearching || !customerId) {
      setSearchItems([]);
      return;
    }

    const ctrl = new AbortController();
    const timer = window.setTimeout(() => {
      setLoading(true);
      setError(undefined);
      void searchCommandPalette({ customer_id: customerId, q: trimmedQuery }, ctrl.signal)
        .then((response) => {
          setSearchItems(response.items ?? []);
          setDegraded(response.degraded === true);
        })
        .catch((err: unknown) => {
          if (ctrl.signal.aborted) {
            return;
          }
          setError(err instanceof Error ? err : new Error(String(err)));
        })
        .finally(() => {
          if (!ctrl.signal.aborted) {
            setLoading(false);
          }
        });
    }, SEARCH_DEBOUNCE_MS);

    return () => {
      ctrl.abort();
      window.clearTimeout(timer);
    };
  }, [customerId, isSearching, open, trimmedQuery]);

  const onSelectItem = useCallback(
    (item: CommandPaletteItem) => {
      setOpen(false);
      if (customerId) {
        void recordCommandPaletteRecent({ customer_id: customerId, item }).catch(() => undefined);
      }
      navigate(normalizeHref(item.href));
    },
    [customerId, navigate],
  );

  const paletteForbidden = error instanceof ApiError && error.status === 403;

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
          <ErrorBlock title="Command palette forbidden" message={error.message} />
        </div>
      ) : error && !paletteForbidden ? (
        <div className="px-3 pb-3">
          <ErrorBlock title="Command palette failed" message={error.message} />
        </div>
      ) : (
        <CommandList aria-label="Command palette results">
          <CommandEmpty>{loading ? 'Loading...' : isSearching ? 'No matches.' : 'No entries.'}</CommandEmpty>
          {isSearching ? (
            <CommandGroup heading="Results">
              {searchItems.map((item) => (
                <CommandItem key={item.id} value={item.id} onSelect={() => onSelectItem(item)}>
                  <span className="min-w-0 flex-1">
                    <span className="block font-medium">{item.label}</span>
                    {item.meta ? (
                      <span className="block truncate text-xs text-muted-foreground">{item.meta}</span>
                    ) : null}
                  </span>
                  <Badge className="shrink-0" variant="outline">
                    {item.kind}
                  </Badge>
                </CommandItem>
              ))}
            </CommandGroup>
          ) : (
            <>
              {recents.length > 0 ? (
                <CommandGroup heading="Recent">
                  {recents.map((item) => (
                    <CommandItem key={item.id} value={item.id} onSelect={() => onSelectItem(item)}>
                      <span className="min-w-0 flex-1">
                        <span className="block font-medium">{item.label}</span>
                        {item.meta ? (
                          <span className="block truncate text-xs text-muted-foreground">{item.meta}</span>
                        ) : null}
                      </span>
                      <Badge className="shrink-0" variant="outline">
                        {item.kind}
                      </Badge>
                    </CommandItem>
                  ))}
                </CommandGroup>
              ) : null}
              {recents.length > 0 && catalogItems.length > 0 ? <CommandSeparator /> : null}
              {catalogItems.length > 0 ? (
                <CommandGroup heading="Routes">
                  {catalogItems.map((item) => (
                    <CommandItem key={item.id} value={item.id} onSelect={() => onSelectItem(item)}>
                      <span className="min-w-0 flex-1">
                        <span className="block font-medium">{item.label}</span>
                        {item.meta ? (
                          <span className="block truncate text-xs text-muted-foreground">{item.meta}</span>
                        ) : null}
                      </span>
                      <Badge className="shrink-0" variant="outline">
                        {item.kind}
                      </Badge>
                    </CommandItem>
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
