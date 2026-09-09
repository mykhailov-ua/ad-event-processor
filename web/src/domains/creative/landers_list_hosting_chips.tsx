import type { LanderHostingFilter } from '@/domains/creative/landers_list_types';
import type { LandersHostingCounts } from '@/domains/creative/use_landers_list_view';
import { FilterChipGroup, type FilterChipTone } from '@/shell/filter_chip_group';

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

const HOSTING_CHIP_TONE: Record<string, FilterChipTone> = {
  '': {
    idle: 'border-border bg-card text-foreground hover:border-foreground/35 hover:bg-accent hover:text-foreground',
    active:
      'border-foreground/40 bg-accent text-foreground hover:border-foreground/55 hover:bg-accent/80',
    countIdle: 'text-muted-foreground group-hover:text-foreground',
    countActive: 'text-muted-foreground',
  },
  external: {
    idle: 'border-border bg-card text-muted-foreground hover:border-primary/40 hover:bg-primary/10 hover:text-foreground',
    active: 'border-primary/40 bg-primary/15 text-foreground hover:border-primary/55',
    countIdle: 'text-muted-foreground group-hover:text-foreground',
    countActive: 'text-muted-foreground',
  },
  hosted: {
    idle: 'border-border bg-card text-muted-foreground hover:border-chart-2/50 hover:bg-chart-2/10 hover:text-foreground',
    active: 'border-chart-2/50 bg-chart-2/15 text-foreground hover:border-chart-2/70',
    countIdle: 'text-muted-foreground group-hover:text-foreground',
    countActive: 'text-muted-foreground',
  },
};

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
    <FilterChipGroup
      ariaLabel="Lander hosting filters"
      className={className}
      onChange={onChange}
      options={options}
      toneMap={HOSTING_CHIP_TONE}
      value={value}
    />
  );
}
