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
  className?: string;
};

const CHIP_IDLE = 'border-border bg-card text-foreground';
const CHIP_ACTIVE = 'border-primary bg-primary text-primary-foreground';

export function FilterChipGroup<T extends string>({
  options,
  value,
  onChange,
  ariaLabel,
  countsLoading = false,
  className,
}: FilterChipGroupProps<T>) {
  return (
    <div className={cn('flex flex-wrap gap-2', className)} role="group" aria-label={ariaLabel}>
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
            className={cn(
              adminKit.buttonShell,
              adminKit.chipPaddingX,
              adminKit.chipInnerGap,
              adminKit.controlRadius,
              'max-w-full whitespace-nowrap border font-medium transition-colors',
              selected ? CHIP_ACTIVE : CHIP_IDLE
            )}
            type="button"
            onClick={() => onChange(option.value)}
          >
            {option.label}
            {countLabel != null ? (
              <span
                className={cn(
                  adminKit.controlRadius,
                  'inline-flex min-w-[1.25rem] items-center justify-center px-2 text-[11px] font-semibold tabular-nums',
                  selected
                    ? 'bg-primary-foreground/20 text-primary-foreground'
                    : 'bg-muted text-muted-foreground'
                )}
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
