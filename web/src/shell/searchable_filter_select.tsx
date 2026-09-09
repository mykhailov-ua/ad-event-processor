import { useLayoutEffect, useMemo, useRef, useState } from 'react';
import { Check, ChevronDown, X } from 'lucide-react';

import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Button } from '@/components/ui/button';
import { adminChrome } from '@/lib/admin_chrome';
import { resolvePopoverAlign } from '@/lib/popover_align';
import { cn } from '@/lib/utils';
import {
  searchableFilterSelectEmptyClass,
  searchableFilterSelectGroupLabelClass,
  searchableFilterSelectListClass,
  searchableFilterSelectOptionClass,
  searchableFilterSelectOptionSelectedClass,
  searchableFilterSelectPopoverClass,
  searchableFilterSelectSearchClass,
  searchableFilterSelectSearchRowClass,
  searchableFilterSelectTriggerClass,
} from '@/shell/searchable_filter_select_classes';

export type SearchableFilterOption = {
  value: string;
  label: string;
};

export type SearchableFilterOptionGroup = {
  id: string;
  label: string;
  options: SearchableFilterOption[];
};

export type SearchableFilterSelectProps = {
  'aria-label': string;
  disabled?: boolean;
  title?: string;
  triggerId?: string;
  options: SearchableFilterOption[];
  groups?: SearchableFilterOptionGroup[];
  value: string;
  onValueChange?: (value: string) => void;
  formatOptionLabel?: (value: string, fallback: string) => string;
  searchPlaceholder?: string;
  searchAriaLabel?: string;
  showSearch?: boolean;
  allowFreeform?: boolean;
  matchPopoverToTrigger?: boolean;
  listClassName?: string;
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

function scrollSelectedFilterOptionIntoView(list: HTMLElement | null) {
  if (!list) {
    return;
  }
  list.querySelector<HTMLElement>('[aria-selected="true"]')?.scrollIntoView({ block: 'nearest' });
}

function optionMatchesQuery(
  option: SearchableFilterOption,
  needle: string,
  formatOptionLabel?: (value: string, fallback: string) => string
): boolean {
  const label = resolveOptionLabel(option, formatOptionLabel).toLowerCase();
  return label.includes(needle) || option.value.toLowerCase().includes(needle);
}

export function SearchableFilterSelect({
  'aria-label': ariaLabel,
  disabled = false,
  title,
  triggerId,
  options,
  groups,
  value,
  onValueChange,
  formatOptionLabel,
  searchPlaceholder,
  searchAriaLabel,
  showSearch = true,
  allowFreeform = false,
  matchPopoverToTrigger = false,
  listClassName,
}: SearchableFilterSelectProps) {
  const triggerRef = useRef<HTMLButtonElement>(null);
  const listRef = useRef<HTMLUListElement>(null);
  const [open, setOpen] = useState(false);
  const [align, setAlign] = useState<'start' | 'end'>('start');
  const [query, setQuery] = useState('');

  const selected = options.find((option) => option.value === value);
  const selectedLabel = selected
    ? resolveOptionLabel(selected, formatOptionLabel)
    : value.trim()
      ? value.trim()
      : '';
  const placeholder = searchPlaceholder ?? 'Search...';
  const freeformQuery = query.trim();
  const canUseFreeform =
    allowFreeform &&
    freeformQuery.length > 0 &&
    !options.some(
      (option) =>
        option.value.toLowerCase() === freeformQuery.toLowerCase() ||
        resolveOptionLabel(option, formatOptionLabel).toLowerCase() === freeformQuery.toLowerCase()
    );

  const filteredGroups = useMemo(() => {
    const needle = query.trim().toLowerCase();
    const sourceGroups =
      groups ??
      [
        {
          id: 'all',
          label: '',
          options,
        },
      ];

    return sourceGroups
      .map((group) => ({
        ...group,
        options: needle
          ? group.options.filter((option) => optionMatchesQuery(option, needle, formatOptionLabel))
          : group.options,
      }))
      .filter((group) => group.options.length > 0);
  }, [formatOptionLabel, groups, options, query]);

  const filteredOptions = useMemo(
    () => filteredGroups.flatMap((group) => group.options),
    [filteredGroups]
  );
  const hasGroupHeaders = filteredGroups.some((group) => group.label.length > 0);

  function renderOption(option: SearchableFilterOption) {
    const label = resolveOptionLabel(option, formatOptionLabel);
    const isSelected = option.value === value;

    return (
      <li key={option.value} role="none">
        <button
          aria-selected={isSelected}
          className={cn(
            searchableFilterSelectOptionClass,
            isSelected && searchableFilterSelectOptionSelectedClass
          )}
          role="option"
          type="button"
          onClick={() => {
            onValueChange?.(option.value);
            setOpen(false);
          }}
        >
          <span className="truncate" title={label}>{label}</span>
          {isSelected ? (
            <Check
              aria-hidden
              className="absolute right-2 size-4 text-foreground"
              strokeWidth={2.5}
            />
          ) : null}
        </button>
      </li>
    );
  }

  useLayoutEffect(() => {
    if (!open) {
      return;
    }
    scrollSelectedFilterOptionIntoView(listRef.current);
  }, [open, value]);

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
        <button
          ref={triggerRef}
          id={triggerId}
          aria-expanded={open}
          aria-haspopup="listbox"
          aria-label={ariaLabel}
          className={searchableFilterSelectTriggerClass}
          disabled={disabled}
          title={title}
          type="button"
        >
          <span className="min-w-0 truncate">{selectedLabel || 'Select...'}</span>
          <ChevronDown aria-hidden className="size-4 shrink-0 opacity-60" />
        </button>
      </PopoverTrigger>
      <PopoverContent
        align={align}
        className={cn(
          matchPopoverToTrigger
            ? 'w-[var(--popover-anchor-width)] max-w-[var(--popover-anchor-width)] p-0'
            : searchableFilterSelectPopoverClass
        )}
        matchTriggerMinWidth={matchPopoverToTrigger}
        panelClassName="p-0"
        panelScroll="inner"
        side="bottom"
      >
        {showSearch ? (
          <div className={searchableFilterSelectSearchRowClass}>
            <input
              aria-label={searchAriaLabel ?? ariaLabel}
              className={searchableFilterSelectSearchClass}
              placeholder={placeholder}
              type="search"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter' && canUseFreeform) {
                  event.preventDefault();
                  onValueChange?.(freeformQuery);
                  setOpen(false);
                }
              }}
            />
            {query ? (
              <Button
                aria-label="Clear search"
                className="h-8 w-8 shrink-0 p-0"
                type="button"
                variant="ghost"
                onClick={() => setQuery('')}
              >
                <X aria-hidden className="size-4" />
              </Button>
            ) : null}
          </div>
        ) : null}
        <ul
          ref={listRef}
          className={cn(searchableFilterSelectListClass, listClassName)}
          role="listbox"
        >
          {canUseFreeform ? (
            <li role="none">
              <button
                className={searchableFilterSelectOptionClass}
                role="option"
                type="button"
                onClick={() => {
                  onValueChange?.(freeformQuery);
                  setOpen(false);
                }}
              >
                <span className="truncate">Use &quot;{freeformQuery}&quot;</span>
              </button>
            </li>
          ) : null}
          {filteredOptions.length === 0 && !canUseFreeform ? (
            <li className={searchableFilterSelectEmptyClass} role="none">No matches</li>
          ) : hasGroupHeaders ? (
            filteredGroups.map((group) => (
              <li key={group.id} role="presentation">
                {group.label ? (
                  <div className={searchableFilterSelectGroupLabelClass}>{group.label}</div>
                ) : null}
                <ul className={adminChrome.menuList} role="group" aria-label={group.label || undefined}>
                  {group.options.map((option) => renderOption(option))}
                </ul>
              </li>
            ))
          ) : (
            filteredOptions.map((option) => renderOption(option))
          )}
        </ul>
      </PopoverContent>
    </Popover>
  );
}
