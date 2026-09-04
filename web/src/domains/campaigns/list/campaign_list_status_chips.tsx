import type { CampaignStatusFilter } from '@/domains/campaigns/list/campaigns_list_types';
import { cn } from '@/lib/utils';

export type CampaignListStatusChipOption = {
  value: CampaignStatusFilter;
  label: string;
  count?: number;
};

export type CampaignListStatusChipsProps = {
  options: CampaignListStatusChipOption[];
  value: CampaignStatusFilter;
  onChange: (value: CampaignStatusFilter) => void;
  countsLoading?: boolean;
  className?: string;
};

const STATUS_CHIP_CLASS: Record<string, { idle: string; active: string; count: string }> = {
  '': {
    idle: 'border-border bg-background text-foreground',
    active: 'border-foreground/40 bg-accent text-foreground',
    count: 'text-muted-foreground',
  },
  ACTIVE: {
    idle: 'border-emerald-500/70 bg-background text-emerald-500',
    active: 'border-emerald-500 bg-emerald-500/15 text-emerald-400',
    count: 'text-emerald-400',
  },
  PAUSED: {
    idle: 'border-amber-500/70 bg-background text-amber-500',
    active: 'border-amber-500 bg-amber-500/15 text-amber-400',
    count: 'text-amber-400',
  },
  ARCHIVED: {
    idle: 'border-border bg-background text-muted-foreground',
    active: 'border-border bg-muted text-foreground',
    count: 'text-muted-foreground',
  },
};

function chipTone(value: CampaignStatusFilter) {
  return STATUS_CHIP_CLASS[value] ?? STATUS_CHIP_CLASS[''];
}

export function CampaignListStatusChips({
  options,
  value,
  onChange,
  countsLoading = false,
  className,
}: CampaignListStatusChipsProps) {
  return (
    <div className={cn('flex flex-wrap gap-2', className)} role="group" aria-label="Campaign status">
      {options.map((option) => {
        const selected = value === option.value;
        const tone = chipTone(option.value);
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
              'inline-flex min-h-7 max-w-full shrink-0 items-center gap-1 whitespace-nowrap rounded-[5px] border px-2 py-1 text-[13px] font-semibold leading-[18px] transition-colors',
              selected ? tone.active : tone.idle,
            )}
            type="button"
            onClick={() => onChange(option.value)}
          >
            {option.label}
            <span className={cn('text-[11px] font-semibold tabular-nums', tone.count)}>{countLabel}</span>
          </button>
        );
      })}
    </div>
  );
}
