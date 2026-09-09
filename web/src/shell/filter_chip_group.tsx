import { cn } from '@/lib/utils';
import { adminKit } from '@/lib/admin_kit';

export type FilterChipOption<T extends string> = {
  value: T;
  label: string;
  count?: number | string;
};

export type FilterChipGroupProps<T extends string> = {
  options: FilterChipOption<T>[];
  value: T;
  onChange: (value: T) => void;
  ariaLabel: string;
  countsLoading?: boolean;
};

const CHIP_IDLE = 'border-border bg-card text-foreground';
const CHIP_ACTIVE = 'border-primary bg-primary text-primary-foreground';

export function FilterChipGroup<T extends string>({
  options,
  value,
  onChange,
  ariaLabel,
  countsLoading = false,
}: FilterChipGroupProps<T>) {
  return (
    <div  role="group" aria-label={ariaLabel}>
      {options.map((option) => {
        const selected = value === option.value;
        const countLabel =
          countsLoading && option.count == null
            ? '...'
            : option.count != null
              ? String(option.count)
              : undefined;

        return (
          <button
            key={option.value || '__all'}
            aria-pressed={selected}
           
            type="button"
            onClick={() => onChange(option.value)}
          >
            {option.label}
            {countLabel != null ? (
              <span
               
              >
                {countLabel}
              </span>
            ) : null}
          </button>
        );
      })}
    </div>
  );
}
