import { Button } from '@/components/ui/button';
import { adminKit } from '@/lib/admin_kit';
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

const CHIP_IDLE =
  'border-border bg-card text-foreground hover:border-foreground/35 hover:bg-accent hover:text-foreground';
const CHIP_ACTIVE = 'border-primary bg-primary text-primary-foreground hover:border-primary hover:bg-primary';

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
          <Button
            key={option.value || 'all'}
            aria-pressed={selected}
            className={cn(
              uiSurfaces.chip,
              adminKit.nestedRadius,
              'font-semibold',
              selected ? CHIP_ACTIVE : CHIP_IDLE
            )}
            type="button"
            variant="outline"
            onClick={() => onChange(option.value)}
          >
            {option.label}
            <span className={cn(uiSurfaces.chipCount, selected ? '' : 'text-muted-foreground')}>
              {countLabel}
            </span>
          </Button>
        );
      })}
    </ChipRow>
  );
}
