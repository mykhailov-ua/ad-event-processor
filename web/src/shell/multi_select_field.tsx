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
    <label className={cn('text-sm font-medium text-zinc-700 dark:text-zinc-300 flex flex-col gap-1 text-sm font-medium', className)} htmlFor={id}>
      {label}
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <button
            id={id}
            aria-expanded={open}
            className="relative w-full flex h-8 w-full items-center justify-between rounded-md border border-zinc-200 bg-white px-3 text-sm dark:border-zinc-700 dark:bg-zinc-950"
            type="button"
          >
            <span className="truncate" title={summary}>
              {summary}
            </span>
            <ChevronDown
              aria-hidden
              className={cn('h-4 w-4 opacity-50', open && 'rotate-180')}
            />
          </button>
        </PopoverTrigger>
        <PopoverContent align="start" className="w-64 p-0" sideOffset={4}>
          <ul className="max-h-64 overflow-y-auto p-1">
            {options.map((option) => {
              const selected = value.includes(option.id);
              const locked = selected && value.length <= minSelected;
              return (
                <li key={option.id}>
                  <label className="flex items-center gap-2 text-sm">
                    <Checkbox
                      checked={selected}
                      disabled={locked}
                      onCheckedChange={(next) => toggleOption(option.id, next === true)}
                    />
                    <span className="text-xs font-medium text-zinc-500 dark:text-zinc-400">{option.label}</span>
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
