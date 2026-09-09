import { ErrorBlock } from '@/shell/error_block';
import { CommandPaletteRow } from '@/shell/command_palette_row';
import { useCommandPalette } from '@/shell/use_command_palette';
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandList,
  CommandSeparator,
} from '@/components/ui/command';

export type CommandPaletteProps = {
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
};

export function CommandPalette({ open: controlledOpen, onOpenChange }: CommandPaletteProps = {}) {
  const {
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
    activeError,
    activeLoading,
    paletteForbidden,
    onSelectItem,
  } = useCommandPalette({ open: controlledOpen, onOpenChange });

  return (
    <CommandDialog onOpenChange={setOpen} open={open} shouldFilter={false}>
      <p  id="command-palette-description">
        Search admin routes and entities. Press Ctrl+K or Cmd+K to reopen.
      </p>
      <CommandInput
        aria-label="Command palette search"
        placeholder="Search routes, campaigns, reports..."
        value={query}
        onValueChange={setQuery}
      />
      {degraded ? (
        <p >
          Search results may be incomplete.
        </p>
      ) : null}
      {paletteForbidden ? (
        <div >
          <ErrorBlock title="Command palette forbidden" message={activeError?.message ?? ''} />
        </div>
      ) : activeError && !paletteForbidden ? (
        <div >
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
              {contextualItems.length > 0 ? (
                <CommandGroup heading="Selection">
                  {contextualItems.map((item) => (
                    <CommandPaletteRow key={item.id} item={item} onSelect={onSelectItem} />
                  ))}
                </CommandGroup>
              ) : null}
              {contextualItems.length > 0 && (recents.length > 0 || catalogItems.length > 0) ? (
                <CommandSeparator />
              ) : null}
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
