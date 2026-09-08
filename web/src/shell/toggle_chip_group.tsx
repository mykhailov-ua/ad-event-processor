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
  className?: string;
};

export function ToggleChipGroup<T extends string>({
  options,
  value,
  onChange,
  countsLoading = false,
  className,
}: ToggleChipGroupProps<T>) {
  return (
    <ChipRow className={className}>
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
            className={cn(
              uiSurfaces.chip,
              selected
                ? 'border-primary bg-primary text-primary-foreground hover:bg-primary/90'
                : 'border-border bg-secondary text-secondary-foreground hover:bg-secondary/80'
            )}
            onClick={() => onChange(option.value)}
            type="button"
          >
            {option.label}
            <span
              className={cn(
                uiSurfaces.chipCount,
                'inline-flex h-5 min-w-5 items-center justify-center rounded-sm px-1.5',
                selected
                  ? 'bg-primary-foreground/20 text-primary-foreground'
                  : 'bg-muted text-muted-foreground'
              )}
            >
              {countLabel}
            </span>
          </button>
        );
      })}
    </ChipRow>
  );
}
