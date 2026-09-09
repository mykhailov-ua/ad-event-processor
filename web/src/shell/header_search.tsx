import { useEffect, useId, useRef, useState, type RefObject } from 'react';

import type { CommandPaletteItem } from '@/api/command_palette_api';
import { Input } from '@/components/ui/input';
import { adminChrome } from '@/lib/admin_chrome';
import { cn } from '@/lib/utils';
import { ErrorBlock } from '@/shell/error_block';
import { useHeaderSearch } from '@/shell/use_header_search';

export type HeaderSearchProps = {
  inputRef?: RefObject<HTMLInputElement | null>;
};

function groupItems(items: CommandPaletteItem[]) {
  const routes: CommandPaletteItem[] = [];
  const entities: CommandPaletteItem[] = [];
  for (const item of items) {
    if (item.kind === 'route') {
      routes.push(item);
      continue;
    }
    entities.push(item);
  }
  return { routes, entities };
}

export function HeaderSearch({ inputRef: externalInputRef }: HeaderSearchProps) {
  const listboxId = useId();
  const internalInputRef = useRef<HTMLInputElement>(null);
  const inputRef = externalInputRef ?? internalInputRef;
  const panelRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);

  const {
    query,
    setQuery,
    items,
    isSearching,
    degraded,
    activeError,
    activeLoading,
    paletteForbidden,
    onSelectItem,
  } = useHeaderSearch();

  useEffect(() => {
    if (!open) {
      return undefined;
    }
    const onPointerDown = (event: MouseEvent) => {
      const target = event.target;
      if (!(target instanceof Node)) {
        return;
      }
      if (inputRef.current?.contains(target) || panelRef.current?.contains(target)) {
        return;
      }
      setOpen(false);
    };
    document.addEventListener('mousedown', onPointerDown);
    return () => document.removeEventListener('mousedown', onPointerDown);
  }, [inputRef, open]);

  const showPanel = open && isSearching;
  const { routes, entities } = groupItems(items);

  return (
    <div className="relative w-full max-w-xl">
      <Input
        ref={inputRef}
        aria-autocomplete="list"
        aria-controls={showPanel ? listboxId : undefined}
        aria-expanded={showPanel}
        aria-label="Search pages and records"
        className="w-full"
        placeholder="Search pages, billing, campaigns..."
        role="combobox"
        type="search"
        value={query}
        onBlur={() => {
          window.setTimeout(() => setOpen(false), 120);
        }}
        onChange={(event) => {
          setQuery(event.target.value);
          setOpen(true);
        }}
        onFocus={() => {
          if (query.trim()) {
            setOpen(true);
          }
        }}
        onKeyDown={(event) => {
          if (event.key === 'Escape') {
            setOpen(false);
            inputRef.current?.blur();
          }
        }}
      />
      {showPanel ? (
        <div
          ref={panelRef}
          className={cn(
            adminChrome.floating,
            'absolute left-0 right-0 top-[calc(100%+0.25rem)] z-[10001] max-h-[min(24rem,calc(100vh-4rem))] overflow-auto p-1'
          )}
          id={listboxId}
          role="listbox"
        >
          {paletteForbidden ? (
            <ErrorBlock message={activeError?.message ?? ''} title="Search forbidden" />
          ) : activeError && !paletteForbidden ? (
            <ErrorBlock message={activeError.message} title="Search failed" />
          ) : (
            <>
              {degraded ? (
                <p className="px-3 py-2 text-[13px] text-muted-foreground">
                  Search results may be incomplete.
                </p>
              ) : null}
              {activeLoading ? (
                <p className="px-3 py-2 text-[13px] text-muted-foreground">Loading...</p>
              ) : items.length === 0 ? (
                <p className="px-3 py-2 text-[13px] text-muted-foreground">No matches.</p>
              ) : (
                <>
                  {routes.length > 0 ? (
                    <section className="py-1">
                      <h3 className="px-3 py-1 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
                        Pages
                      </h3>
                      {routes.map((item) => (
                        <button
                          key={item.id}
                          className={adminChrome.menuItem}
                          role="option"
                          type="button"
                          onMouseDown={(event) => event.preventDefault()}
                          onClick={() => onSelectItem(item)}
                        >
                          <span className="block font-medium">{item.label}</span>
                          {item.meta ? (
                            <span className="block text-[12px] text-muted-foreground">{item.meta}</span>
                          ) : null}
                        </button>
                      ))}
                    </section>
                  ) : null}
                  {entities.length > 0 ? (
                    <section className="py-1">
                      <h3 className="px-3 py-1 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
                        Records
                      </h3>
                      {entities.map((item) => (
                        <button
                          key={item.id}
                          className={adminChrome.menuItem}
                          role="option"
                          type="button"
                          onMouseDown={(event) => event.preventDefault()}
                          onClick={() => onSelectItem(item)}
                        >
                          <span className="block font-medium">{item.label}</span>
                          {item.meta ? (
                            <span className="block text-[12px] text-muted-foreground">{item.meta}</span>
                          ) : null}
                        </button>
                      ))}
                    </section>
                  ) : null}
                </>
              )}
            </>
          )}
        </div>
      ) : null}
    </div>
  );
}
