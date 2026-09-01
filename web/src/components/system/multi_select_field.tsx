import { useMemo, useState, type WheelEvent } from 'react';
import { Check, ChevronDown, X } from 'lucide-react';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command';
import { Label } from '@/components/ui/label';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { cn } from '@/lib/utils';

export type MultiSelectOption<T extends string> = {
  id: T;
  label: string;
};

export type MultiSelectFieldProps<T extends string> = {
  id: string;
  label: string;
  options: MultiSelectOption<T>[];
  value: T[];
  onChange: (value: T[]) => void;
  className?: string;
  labelTone?: 'field' | 'nested';
  minSelected?: number;
};

function stopDialogWheelPropagation(event: WheelEvent<HTMLElement>) {
  event.stopPropagation();
}

export function MultiSelectField<T extends string>({
  id,
  label,
  options,
  value,
  onChange,
  className,
  labelTone = 'field',
  minSelected = 1,
}: MultiSelectFieldProps<T>) {
  const [open, setOpen] = useState(false);
  const optionById = useMemo(() => new Map(options.map((option) => [option.id, option])), [options]);

  function toggleOption(optionId: T) {
    if (value.includes(optionId)) {
      if (value.length <= minSelected) {
        return;
      }
      onChange(value.filter((item) => item !== optionId));
      return;
    }
    onChange([...value, optionId]);
  }

  function removeOption(optionId: T) {
    if (value.length <= minSelected) {
      return;
    }
    onChange(value.filter((item) => item !== optionId));
  }

  function clearAll() {
    if (options.length === 0) {
      return;
    }
    onChange([options[0].id]);
  }

  return (
    <div className={cn('grid gap-3', className)}>
      <Label
        htmlFor={id}
        className={cn(
          labelTone === 'nested'
            ? 'text-xs font-medium text-muted-foreground'
            : 'text-sm font-medium text-foreground',
        )}
      >
        {label}
      </Label>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            id={id}
            type="button"
            variant="outline"
            shape="square"
            aria-expanded={open}
            className={cn(
              'ui-preferences-field-trigger h-auto min-h-10 w-full items-start justify-between border px-3 py-2 font-normal shadow-none',
              open && 'border-border/80 ring-1 ring-border/50',
            )}
          >
            <span
              className="ui-scrollbar flex max-h-32 min-w-0 flex-1 flex-wrap content-start gap-2.5 overflow-y-auto overscroll-y-contain text-left"
              onWheel={stopDialogWheelPropagation}
            >
              {value.length === 0 ? (
                <span className="text-muted-foreground">Select items</span>
              ) : (
                value.map((optionId) => (
                  <Badge
                    key={optionId}
                    variant="outline"
                    className="ui-preferences-chip max-w-full shrink-0 gap-1 rounded-md px-2 py-0.5 text-xs font-normal"
                  >
                    <span className="truncate">{optionById.get(optionId)?.label ?? optionId}</span>
                    <button
                      type="button"
                      className="rounded-sm opacity-70 hover:opacity-100"
                      onClick={(event) => {
                        event.stopPropagation();
                        removeOption(optionId);
                      }}
                      aria-label={`Remove ${optionById.get(optionId)?.label ?? optionId}`}
                    >
                      <X className="h-3 w-3" />
                    </button>
                  </Badge>
                ))
              )}
            </span>
            <span className="ml-2 mt-0.5 flex shrink-0 items-center gap-1.5 self-start text-muted-foreground">
              {value.length > minSelected ? (
                <button
                  type="button"
                  className="rounded-sm p-0.5 text-muted-foreground hover:text-foreground"
                  onClick={(event) => {
                    event.stopPropagation();
                    clearAll();
                  }}
                  aria-label="Clear selection"
                >
                  <X className="h-3.5 w-3.5" />
                </button>
              ) : null}
              <ChevronDown className={cn('h-4 w-4 transition-transform', open && 'rotate-180')} />
            </span>
          </Button>
        </PopoverTrigger>
        <PopoverContent
          className="ui-preferences-menu z-[60] p-0"
          align="start"
          panelScroll="inner"
          sideOffset={4}
          collisionPadding={20}
        >
          <Command className="flex flex-col overflow-hidden bg-transparent">
            <CommandInput className="h-9 py-2" placeholder="Search..." />
            <CommandList className="ui-preferences-menu-list ui-scrollbar max-h-48 overflow-y-auto p-2">
              <CommandEmpty>No matches.</CommandEmpty>
              <CommandGroup className="p-0">
                {options.map((option) => {
                  const selected = value.includes(option.id);
                  return (
                    <CommandItem
                      key={option.id}
                      value={option.label}
                      className={cn(
                        'rounded-md px-2 py-0.5 text-sm text-muted-foreground',
                        selected && 'bg-[hsl(var(--prefs-menu-item))] text-foreground',
                      )}
                      onSelect={() => toggleOption(option.id)}
                    >
                      <Check
                        className={cn(
                          'mr-2 h-4 w-4',
                          selected ? 'text-foreground opacity-100' : 'opacity-0',
                        )}
                      />
                      {option.label}
                    </CommandItem>
                  );
                })}
              </CommandGroup>
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
    </div>
  );
}
