import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

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
    <div className={cn('flex flex-wrap gap-2', className)}>
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
            className="gap-1.5 px-3 text-xs"
            onClick={() => onChange(option.value)}
            type="button"
            variant={selected ? 'default' : 'secondary'}
          >
            {option.label}
            <span
              className={cn(
                'inline-flex h-5 min-w-5 items-center justify-center rounded-[var(--admin-radius-sm)] px-1.5 text-[11px] font-medium tabular-nums',
                selected
                  ? 'bg-primary-foreground text-primary'
                  : 'bg-muted text-muted-foreground',
              )}
            >
              {countLabel}
            </span>
          </Button>
        );
      })}
    </div>
  );
}
