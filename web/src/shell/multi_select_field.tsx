import { useMemo, useState } from 'react';
import { ChevronDown } from 'lucide-react';

import { Checkbox } from '@/components/ui/checkbox';
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
  minSelected?: number;
};

function formatSelectionSummary<T extends string>(
  value: T[],
  optionById: Map<T, MultiSelectOption<T>>,
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
  className,
  minSelected = 1,
}: MultiSelectFieldProps<T>) {
  const [open, setOpen] = useState(false);
  const optionById = useMemo(() => new Map(options.map((option) => [option.id, option])), [options]);
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
    <label className={cn('admin-label admin-label--stacked', className)} htmlFor={id}>
      {label}
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <button
            id={id}
            aria-expanded={open}
            className="admin-select admin-select-trigger admin-multi-select-trigger"
            type="button"
          >
            <span className="truncate" title={summary}>
              {summary}
            </span>
            <ChevronDown
              aria-hidden
              className={cn('admin-multi-select-chevron', open && 'is-open')}
            />
          </button>
        </PopoverTrigger>
        <PopoverContent align="start" className="admin-multi-select-menu p-0" sideOffset={4}>
          <ul className="admin-multi-select-menu__list">
            {options.map((option) => {
              const selected = value.includes(option.id);
              const locked = selected && value.length <= minSelected;
              return (
                <li key={option.id}>
                  <label className="admin-columns-menu__item">
                    <Checkbox
                      checked={selected}
                      disabled={locked}
                      onCheckedChange={(next) => toggleOption(option.id, next === true)}
                    />
                    <span className="admin-columns-menu__label">{option.label}</span>
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
