import type { CommandPaletteItem } from '@/api/command_palette_api';
import { CommandItem, CommandShortcut } from '@/components/ui/command';

export type CommandPaletteRowProps = {
  item: CommandPaletteItem;
  onSelect: (item: CommandPaletteItem) => void;
};

export function CommandPaletteRow({ item, onSelect }: CommandPaletteRowProps) {
  return (
    <CommandItem value={item.id} onSelect={() => onSelect(item)}>
      <span className="min-w-0 flex-1">
        <span className="block truncate font-medium leading-snug">{item.label}</span>
        {item.meta ? (
          <span className="block whitespace-normal text-xs leading-snug text-muted-foreground">
            {item.meta}
          </span>
        ) : null}
      </span>
      <CommandShortcut data-command-meta className="capitalize">
        {item.kind}
      </CommandShortcut>
    </CommandItem>
  );
}
