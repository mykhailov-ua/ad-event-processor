import { Button } from '@/components/ui/button';
import { adminKit } from '@/lib/admin_kit';
import { uiSurfaces } from '@/lib/ui_surfaces';
import { cn } from '@/lib/utils';

export type FilterChipOption<T extends string> = {
  value: T;
  label: string;
  count?: number | string;
};

export type FilterChipTone = {
  idle: string;
  active: string;
  countIdle: string;
  countActive: string;
};

export type FilterChipGroupProps<T extends string> = {
  options: FilterChipOption<T>[];
  value: T;
  onChange: (value: T) => void;
  ariaLabel: string;
  countsLoading?: boolean;
  className?: string;
  toneMap?: Record<string, FilterChipTone>;
  defaultCountLabel?: string;
};

const CHIP_IDLE =
  'border-border bg-card text-foreground hover:border-foreground/35 hover:bg-accent hover:text-foreground';
const CHIP_ACTIVE =
  'border-primary bg-primary text-primary-foreground hover:border-primary hover:bg-primary';

function resolveChipTone<T extends string>(
  option: FilterChipOption<T>,
  toneMap?: Record<string, FilterChipTone>
): FilterChipTone | undefined {
  return toneMap?.[option.value] ?? toneMap?.[''];
}

export function FilterChipGroup<T extends string>({
  options,
  value,
  onChange,
  ariaLabel,
  countsLoading = false,
  className,
  toneMap,
  defaultCountLabel,
}: FilterChipGroupProps<T>) {
  return (
    <div className={cn(uiSurfaces.chipRow, className)} role="group" aria-label={ariaLabel}>
      {options.map((option) => {
        const selected = value === option.value;
        const tone = resolveChipTone(option, toneMap);
        const countLabel =
          countsLoading && option.count == null
            ? '...'
            : option.count != null
              ? String(option.count)
              : defaultCountLabel;

        const chipClassName = cn(
          toneMap ? 'group' : undefined,
          uiSurfaces.chip,
          adminKit.nestedRadius,
          toneMap ? undefined : 'font-semibold',
          tone ? (selected ? tone.active : tone.idle) : selected ? CHIP_ACTIVE : CHIP_IDLE
        );

        if (toneMap) {
          return (
            <button
              key={option.value || '__all'}
              aria-pressed={selected}
              className={chipClassName}
              type="button"
              onClick={() => onChange(option.value)}
            >
              {option.label}
              {countLabel != null ? (
                <span
                  className={cn(
                    uiSurfaces.chipCount,
                    tone ? (selected ? tone.countActive : tone.countIdle) : undefined
                  )}
                >
                  {countLabel}
                </span>
              ) : null}
            </button>
          );
        }

        return (
          <Button
            key={option.value || '__all'}
            aria-pressed={selected}
            className={chipClassName}
            type="button"
            variant="outline"
            onClick={() => onChange(option.value)}
          >
            {option.label}
            {countLabel != null ? (
              <span className={cn(uiSurfaces.chipCount, selected ? '' : 'text-muted-foreground')}>
                {countLabel}
              </span>
            ) : null}
          </Button>
        );
      })}
    </div>
  );
}
