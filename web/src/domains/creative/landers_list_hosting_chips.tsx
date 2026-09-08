import type { LanderHostingFilter } from '@/domains/creative/landers_list_types';
import type { LandersHostingCounts } from '@/domains/creative/use_landers_list_view';
import { uiSurfaces } from '@/lib/ui_surfaces';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';
import { ChipRow } from '@/shell/ui_bands';

export type LandersListHostingChipOption = {
  value: LanderHostingFilter;
  label: string;
  count?: number;
};

export type LandersListHostingChipsProps = {
  options: LandersListHostingChipOption[];
  value: LanderHostingFilter;
  onChange: (value: LanderHostingFilter) => void;
  className?: string;
};

const CHIP_TONE: Record<
  string,
  { idle: string; active: string; countIdle: string; countActive: string }
> = {
  '': {
    idle:
      'border-border bg-card text-foreground hover:border-foreground/35 hover:bg-accent hover:text-foreground',
    active:
      'border-foreground/40 bg-accent text-foreground hover:border-foreground/55 hover:bg-accent/80',
    countIdle: 'text-muted-foreground group-hover:text-foreground',
    countActive: 'text-muted-foreground',
  },
  external: {
    idle:
      'border-border bg-card text-muted-foreground hover:border-primary/40 hover:bg-primary/10 hover:text-foreground',
    active: 'border-primary/40 bg-primary/15 text-foreground hover:border-primary/55',
    countIdle: 'text-muted-foreground group-hover:text-foreground',
    countActive: 'text-muted-foreground',
  },
  hosted: {
    idle:
      'border-border bg-card text-muted-foreground hover:border-chart-2/50 hover:bg-chart-2/10 hover:text-foreground',
    active: 'border-chart-2/50 bg-chart-2/15 text-foreground hover:border-chart-2/70',
    countIdle: 'text-muted-foreground group-hover:text-foreground',
    countActive: 'text-muted-foreground',
  },
};

function chipTone(value: LanderHostingFilter) {
  return CHIP_TONE[value] ?? CHIP_TONE[''];
}

export function buildLandersHostingChipOptions(
  counts: LandersHostingCounts
): LandersListHostingChipOption[] {
  return [
    { value: '', label: 'All', count: counts.total },
    { value: 'external', label: 'External', count: counts.external },
    { value: 'hosted', label: 'Hosted', count: counts.hosted },
  ];
}

export function LandersListHostingChips({
  options,
  value,
  onChange,
  className,
}: LandersListHostingChipsProps) {
  return (
    <ChipRow aria-label="Lander hosting filters" className={className} role="group">
      {options.map((option) => {
        const selected = value === option.value;
        const tone = chipTone(option.value);

        return (
          <button
            key={option.value || 'all'}
            aria-pressed={selected}
            className={cn(
              'group',
              uiSurfaces.chip,
              adminKit.controlRadius,
              selected ? tone.active : tone.idle
            )}
            type="button"
            onClick={() => onChange(option.value)}
          >
            {option.label}
            {option.count != null ? (
              <span
                className={cn(
                  uiSurfaces.chipCount,
                  selected ? tone.countActive : tone.countIdle
                )}
              >
                {option.count}
              </span>
            ) : null}
          </button>
        );
      })}
    </ChipRow>
  );
}
