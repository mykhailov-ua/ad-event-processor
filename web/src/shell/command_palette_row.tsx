import type { CommandPaletteItem } from '@/api/command_palette_api';
import { CommandItem, CommandShortcut } from '@/components/ui/command';

export type CommandPaletteRowProps = {
  item: CommandPaletteItem;
  onSelect: (item: CommandPaletteItem) => void;
};

export function CommandPaletteRow({ item, onSelect }: CommandPaletteRowProps) {
  return (
    <CommandItem value={item.id} onSelect={() => onSelect(item)}>
      <span className="min-w-0 flex-1" >
        <span className="block font-medium" >{item.label}</span>
        {item.meta ? (
          <span className="block whitespace-normal text-xs text-muted-foreground" >
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
