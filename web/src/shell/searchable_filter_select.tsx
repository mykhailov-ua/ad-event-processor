import { useMemo, useRef, useState } from 'react';
import { Check, ChevronDown, Search, X } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { resolvePopoverAlign } from '@/lib/popover_align';
import { cn } from '@/lib/utils';
import {
  searchableFilterSelectListClass,
  searchableFilterSelectOptionClass,
  searchableFilterSelectOptionSelectedClass,
  searchableFilterSelectPopoverClass,
  searchableFilterSelectSearchClass,
  searchableFilterSelectTriggerClass,
} from '@/shell/searchable_filter_select_classes';

export type SearchableFilterOption = {
  value: string;
  label: string;
};

export type SearchableFilterSelectProps = {
  'aria-label': string;
  className?: string;
  disabled?: boolean;
  title?: string;
  triggerId?: string;
  options: SearchableFilterOption[];
  value: string;
  onValueChange?: (value: string) => void;
  formatOptionLabel?: (value: string, fallback: string) => string;
  searchPlaceholder?: string;
  searchAriaLabel?: string;
};

function resolveOptionLabel(
  option: SearchableFilterOption | undefined,
  formatOptionLabel?: (value: string, fallback: string) => string
): string {
  if (!option) {
    return '';
  }
  if (formatOptionLabel) {
    return formatOptionLabel(option.value, option.label);
  }
  return option.label;
}

export function SearchableFilterSelect({
  'aria-label': ariaLabel,
  className,
  disabled = false,
  title,
  triggerId,
  options,
  value,
  onValueChange,
  formatOptionLabel,
  searchPlaceholder,
  searchAriaLabel,
}: SearchableFilterSelectProps) {
  const triggerRef = useRef<HTMLButtonElement>(null);
  const [open, setOpen] = useState(false);
  const [align, setAlign] = useState<'start' | 'end'>('start');
  const [query, setQuery] = useState('');

  const selected = options.find((option) => option.value === value) ?? options[0];
  const selectedLabel = resolveOptionLabel(selected, formatOptionLabel);
  const placeholder = searchPlaceholder ?? selectedLabel ?? 'Search...';

  const filteredOptions = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) {
      return options;
    }
    return options.filter((option) => {
      const label = resolveOptionLabel(option, formatOptionLabel).toLowerCase();
      return label.includes(needle) || option.value.toLowerCase().includes(needle);
    });
  }, [formatOptionLabel, options, query]);

  function handleOpenChange(nextOpen: boolean) {
    if (nextOpen) {
      setAlign(resolvePopoverAlign(triggerRef.current));
      setQuery('');
    }
    setOpen(nextOpen);
  }

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        <Button
          ref={triggerRef}
          id={triggerId}
          aria-expanded={open}
          aria-haspopup="listbox"
          aria-label={ariaLabel}
          className={cn(
            searchableFilterSelectTriggerClass,
            'h-auto font-normal shadow-none',
            className
          )}
          disabled={disabled}
          title={title}
          type="button"
          variant="outline"
        >
          <span className="min-w-0 flex-1 truncate text-left">{selectedLabel}</span>
          <ChevronDown className="h-4 w-4 shrink-0 opacity-60" aria-hidden />
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align={align}
        className={cn(searchableFilterSelectPopoverClass, 'p-0')}
        side="bottom"
      >
        <div className={searchableFilterSelectSearchClass}>
          <Search className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden />
          <input
            aria-label={searchAriaLabel ?? ariaLabel}
            className="min-w-0 flex-1 bg-transparent text-[13px] leading-[18px] text-foreground outline-none placeholder:text-muted-foreground"
            placeholder={placeholder}
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
          />
          {query ? (
            <Button
              aria-label="Clear search"
              className="h-7 w-7 shrink-0 text-muted-foreground shadow-none hover:bg-transparent hover:text-foreground"
              type="button"
              variant="ghost"
              onClick={() => setQuery('')}
            >
              <X className="h-4 w-4" aria-hidden />
            </Button>
          ) : null}
        </div>
        <ul className={searchableFilterSelectListClass} role="listbox">
          {filteredOptions.length === 0 ? (
            <li className="px-3 py-2 text-[13px] leading-[18px] text-muted-foreground" role="none">
              No matches
            </li>
          ) : (
            filteredOptions.map((option) => {
              const label = resolveOptionLabel(option, formatOptionLabel);
              const isSelected = option.value === value;
              return (
                <li key={option.value} role="none">
                  <Button
                    aria-selected={isSelected}
                    className={cn(
                      searchableFilterSelectOptionClass,
                      'h-auto justify-between font-normal shadow-none',
                      isSelected && searchableFilterSelectOptionSelectedClass
                    )}
                    role="option"
                    type="button"
                    variant="ghost"
                    onClick={() => {
                      onValueChange?.(option.value);
                      setOpen(false);
                    }}
                  >
                    <span className="truncate" title={label}>
                      {label}
                    </span>
                    {isSelected ? (
                      <Check className="h-4 w-4 shrink-0 text-foreground" aria-hidden />
                    ) : null}
                  </Button>
                </li>
              );
            })
          )}
        </ul>
      </PopoverContent>
    </Popover>
  );
}
