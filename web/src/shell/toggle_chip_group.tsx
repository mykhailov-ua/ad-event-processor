import { uiSurfaces } from '@/lib/ui_surfaces';
import { cn } from '@/lib/utils';
import { ChipRow } from '@/shell/ui_bands';

export type ToggleChipOption<T extends string> = {
  value: T;
  label: string;
  count?: number;
};

export type ToggleChipGroupProps<T extends string> = {
  options: ToggleChipOption<T>[];
  value: T;
  onChange: (value: T) => void;
  countsLoading?: boolean;
};

export function ToggleChipGroup<T extends string>({
  options,
  value,
  onChange,
  countsLoading = false,
}: ToggleChipGroupProps<T>) {
  return (
    <ChipRow >
      {options.map((option) => {
        const selected = value === option.value;
        const countLabel =
          countsLoading && option.count == null
            ? '...'
            : option.count != null
              ? String(option.count)
              : '0';

        return (
          <button
            key={option.value || 'all'}
            aria-pressed={selected}
           
            onClick={() => onChange(option.value)}
            type="button"
          >
            {option.label}
            <span
             
            >
              {countLabel}
            </span>
          </button>
        );
      })}
    </ChipRow>
  );
}
