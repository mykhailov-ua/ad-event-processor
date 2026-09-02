import { useRef, useState } from 'react';
import { Check, ChevronsUpDown } from 'lucide-react';

import { Button } from '@/components/ui/button';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { resolvePopoverAlign } from '@/lib/popover_align';
import { cn } from '@/lib/utils';

export type CustomerComboboxOption = {
  id: string;
  name: string;
};

export type CustomerComboboxProps = {
  id?: string;
  value: string;
  options: CustomerComboboxOption[];
  loading?: boolean;
  disabled?: boolean;
  onValueChange: (customerId: string) => void;
};

export function CustomerCombobox({
  id,
  value,
  options,
  loading = false,
  disabled = false,
  onValueChange,
}: CustomerComboboxProps) {
  const triggerRef = useRef<HTMLButtonElement>(null);
  const [open, setOpen] = useState(false);
  const [align, setAlign] = useState<'start' | 'end'>('start');
  const selected = options.find((option) => option.id === value);

  const triggerLabel = loading
    ? 'Loading customers...'
    : selected
      ? selected.name
      : value
        ? value
        : 'All customers';

  return (
    <Popover
      open={open}
      onOpenChange={(nextOpen) => {
        if (nextOpen) {
          setAlign(resolvePopoverAlign(triggerRef.current));
        }
        setOpen(nextOpen);
      }}
    >
      <PopoverTrigger asChild>
        <Button
          ref={triggerRef}
          id={id}
          type="button"
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-full min-w-0 justify-between text-sm font-normal"
          disabled={disabled || loading}
        >
          <span className="min-w-0 flex-1 truncate text-left">{triggerLabel}</span>
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="p-0" align={align}>
        <Command>
          <CommandInput placeholder="Search customer..." />
          <CommandList>
            <CommandEmpty>No customer found.</CommandEmpty>
            <CommandGroup>
              <CommandItem
                value="all customers"
                onSelect={() => {
                  onValueChange('');
                  setOpen(false);
                }}
              >
                <Check
                  className={cn('h-4 w-4', value === '' ? 'opacity-100' : 'opacity-0')}
                />
                All customers
              </CommandItem>
              {options.map((customer) => (
                <CommandItem
                  key={customer.id}
                  value={`${customer.name} ${customer.id}`}
                  onSelect={() => {
                    onValueChange(customer.id);
                    setOpen(false);
                  }}
                >
                  <Check
                    className={cn(
                      'h-4 w-4',
                      value === customer.id ? 'opacity-100' : 'opacity-0',
                    )}
                  />
                  <span className="truncate">{customer.name}</span>
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
