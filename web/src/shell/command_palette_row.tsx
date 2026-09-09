import type { CommandPaletteItem } from '@/api/command_palette_api';
import { CommandItem, CommandShortcut } from '@/components/ui/command';

export type CommandPaletteRowProps = {
  item: CommandPaletteItem;
  onSelect: (item: CommandPaletteItem) => void;
};

export function CommandPaletteRow({ item, onSelect }: CommandPaletteRowProps) {
  return (
    <CommandItem value={item.id} onSelect={() => onSelect(item)}>
      <span >
        <span >{item.label}</span>
        {item.meta ? (
          <span >
            {item.meta}
          </span>
        ) : null}
      </span>
      <CommandShortcut data-command-meta >
        {item.kind}
      </CommandShortcut>
    </CommandItem>
  );
}
