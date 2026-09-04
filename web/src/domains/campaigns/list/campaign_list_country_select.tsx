import { useMemo, useRef, useState } from 'react';
import { Check, ChevronDown, Search, X } from 'lucide-react';

import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import type { CampaignsListFilterOption } from '@/domains/campaigns/list/campaigns_list_filter_select';
import { resolvePopoverAlign } from '@/lib/popover_align';
import { cn } from '@/lib/utils';

const ALL_OPTION_VALUE = '__all__';

const regionNames =
  typeof Intl !== 'undefined' && 'DisplayNames' in Intl
    ? new Intl.DisplayNames(['en'], { type: 'region' })
    : null;

export function campaignCountryOptionLabel(value: string, fallback: string): string {
  if (value === ALL_OPTION_VALUE) {
    return 'All countries';
  }
  if (!regionNames) {
    return fallback;
  }
  try {
    return regionNames.of(value) ?? fallback;
  } catch {
    return fallback;
  }
}

export type CampaignListCountrySelectProps = {
  'aria-label': string;
  className?: string;
  disabled?: boolean;
  options: CampaignsListFilterOption[];
  title?: string;
  value: string;
  onValueChange?: (value: string) => void;
};

export function CampaignListCountrySelect({
  'aria-label': ariaLabel,
  className,
  disabled = false,
  options,
  title,
  value,
  onValueChange,
}: CampaignListCountrySelectProps) {
  const triggerRef = useRef<HTMLButtonElement>(null);
  const [open, setOpen] = useState(false);
  const [align, setAlign] = useState<'start' | 'end'>('start');
  const [query, setQuery] = useState('');

  const selected = options.find((option) => option.value === value) ?? options[0];
  const selectedLabel = selected
    ? campaignCountryOptionLabel(selected.value, selected.label)
    : 'All countries';

  const filteredOptions = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) {
      return options;
    }
    return options.filter((option) => {
      const label = campaignCountryOptionLabel(option.value, option.label).toLowerCase();
      return label.includes(needle) || option.value.toLowerCase().includes(needle);
    });
  }, [options, query]);

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
          aria-expanded={open}
          aria-haspopup="listbox"
          aria-label={ariaLabel}
          className={cn(
            'campaign-country-select__trigger',
            className,
          )}
          disabled={disabled}
          title={title}
          type="button"
        >
          <span className="truncate">{selectedLabel}</span>
          <ChevronDown className="h-4 w-4 shrink-0 opacity-60" aria-hidden />
        </button>
      </PopoverTrigger>
      <PopoverContent align={align} className="campaign-country-select p-0" side="bottom">
        <div className="campaign-country-select__search">
          <Search className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden />
          <input
            aria-label="Search countries"
            className="min-w-0 flex-1 bg-transparent text-[13px] leading-[18px] text-foreground outline-none placeholder:text-muted-foreground"
            placeholder="All countries"
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
          />
          {query ? (
            <button
              aria-label="Clear search"
              className="shrink-0 text-muted-foreground hover:text-foreground"
              type="button"
              onClick={() => setQuery('')}
            >
              <X className="h-4 w-4" aria-hidden />
            </button>
          ) : null}
        </div>
        <ul className="campaign-country-select__list" role="listbox">
          {filteredOptions.map((option) => {
            const label = campaignCountryOptionLabel(option.value, option.label);
            const isSelected = option.value === value;
            return (
              <li key={option.value} role="none">
                <button
                  aria-selected={isSelected}
                  className={cn(
                    'campaign-country-select__option',
                    isSelected && 'campaign-country-select__option--selected',
                  )}
                  role="option"
                  type="button"
                  onClick={() => {
                    onValueChange?.(option.value);
                    setOpen(false);
                  }}
                >
                  <span className="truncate">{label}</span>
                  {isSelected ? (
                    <Check className="h-4 w-4 shrink-0 text-foreground" aria-hidden />
                  ) : null}
                </button>
              </li>
            );
          })}
        </ul>
      </PopoverContent>
    </Popover>
  );
}
