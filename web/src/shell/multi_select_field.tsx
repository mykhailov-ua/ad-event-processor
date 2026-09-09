import { useMemo, useState } from 'react';
import { ChevronDown } from 'lucide-react';

import { Checkbox } from '@/components/ui/checkbox';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { adminChrome } from '@/lib/admin_chrome';
import { adminKit } from '@/lib/admin_kit';
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
  minSelected?: number;
};

function formatSelectionSummary<T extends string>(
  value: T[],
  optionById: Map<T, MultiSelectOption<T>>
): string {
  if (value.length === 0) {
    return 'None';
  }
  return value.map((optionId) => optionById.get(optionId)?.label ?? optionId).join(', ');
}

export function MultiSelectField<T extends string>({
  id,
  label,
  options,
  value,
  onChange,
  minSelected = 1,
}: MultiSelectFieldProps<T>) {
  const [open, setOpen] = useState(false);
  const optionById = useMemo(
    () => new Map(options.map((option) => [option.id, option])),
    [options]
  );
  const summary = formatSelectionSummary(value, optionById);

  function toggleOption(optionId: T, checked: boolean) {
    if (checked) {
      if (value.includes(optionId)) {
        return;
      }
      onChange([...value, optionId]);
      return;
    }
    if (value.length <= minSelected) {
      return;
    }
    onChange(value.filter((item) => item !== optionId));
  }

  return (
    <label
     
      htmlFor={id}
    >
      {label}
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <button
            id={id}
            aria-expanded={open}
           
            type="button"
          >
            <span  title={summary}>
              {summary}
            </span>
            <ChevronDown aria-hidden  />
          </button>
        </PopoverTrigger>
        <PopoverContent align="start" sideOffset={4}>
          <ul >
            {options.map((option) => {
              const selected = value.includes(option.id);
              return (
                <li key={option.id}>
                  <label >
                    <Checkbox
                      checked={selected}
                      onCheckedChange={(next) => toggleOption(option.id, next === true)}
                    />
                    <span >{option.label}</span>
                  </label>
                </li>
              );
            })}
          </ul>
        </PopoverContent>
      </Popover>
    </label>
  );
}
