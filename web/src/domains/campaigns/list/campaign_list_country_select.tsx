import { useMemo, useRef, useState } from 'react';
import { Check, ChevronDown, Search, X } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import type { CampaignsListFilterOption } from '@/domains/campaigns/list/campaigns_list_filter_select';
import {
  campaignCountrySelectListClass,
  campaignCountrySelectOptionClass,
  campaignCountrySelectOptionSelectedClass,
  campaignCountrySelectPopoverClass,
  campaignCountrySelectSearchClass,
  campaignCountrySelectTriggerClass,
} from '@/domains/campaigns/list/campaign_list_classes';
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
        <Button
          ref={triggerRef}
          aria-expanded={open}
          aria-haspopup="listbox"
          aria-label={ariaLabel}
          className={cn(
            campaignCountrySelectTriggerClass,
            'h-auto font-normal shadow-none',
            className,
          )}
          disabled={disabled}
          title={title}
          type="button"
          variant="outline"
        >
          <span className="whitespace-nowrap">{selectedLabel}</span>
          <ChevronDown className="h-4 w-4 shrink-0 opacity-60" aria-hidden />
        </Button>
      </PopoverTrigger>
      <PopoverContent align={align} className={cn(campaignCountrySelectPopoverClass, 'p-0')} side="bottom">
        <div className={campaignCountrySelectSearchClass}>
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
        <ul className={campaignCountrySelectListClass} role="listbox">
          {filteredOptions.map((option) => {
            const label = campaignCountryOptionLabel(option.value, option.label);
            const isSelected = option.value === value;
            return (
              <li key={option.value} role="none">
                <Button
                  aria-selected={isSelected}
                  className={cn(
                    campaignCountrySelectOptionClass,
                    'h-auto justify-between font-normal shadow-none',
                    isSelected && campaignCountrySelectOptionSelectedClass,
                  )}
                  role="option"
                  type="button"
                  variant="ghost"
                  onClick={() => {
                    onValueChange?.(option.value);
                    setOpen(false);
                  }}
                >
                  <span className="truncate" title={label}>{label}</span>
                  {isSelected ? (
                    <Check className="h-4 w-4 shrink-0 text-foreground" aria-hidden />
                  ) : null}
                </Button>
              </li>
            );
          })}
        </ul>
      </PopoverContent>
    </Popover>
  );
}
