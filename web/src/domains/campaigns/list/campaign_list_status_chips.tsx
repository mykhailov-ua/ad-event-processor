import type { CampaignStatusFilter } from '@/domains/campaigns/list/campaigns_list_types';
import { uiSurfaces } from '@/lib/ui_surfaces';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';
import { ChipRow } from '@/shell/ui_bands';

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

const STATUS_CHIP_CLASS: Record<
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
  ACTIVE: {
    idle:
      'border-border bg-card text-muted-foreground hover:border-admin-status-active hover:bg-admin-status-active/15 hover:text-admin-status-active',
    active:
      'border-admin-status-active bg-admin-status-active/20 text-admin-status-active hover:border-admin-status-active hover:bg-admin-status-active/30',
    countIdle: 'text-muted-foreground group-hover:text-admin-status-active',
    countActive: 'text-admin-status-active',
  },
  PAUSED: {
    idle:
      'border-border bg-card text-muted-foreground hover:border-admin-status-paused hover:bg-admin-status-paused/15 hover:text-admin-status-paused',
    active:
      'border-admin-status-paused bg-admin-status-paused/20 text-admin-status-paused hover:border-admin-status-paused hover:bg-admin-status-paused/30',
    countIdle: 'text-muted-foreground group-hover:text-admin-status-paused',
    countActive: 'text-admin-status-paused',
  },
  ARCHIVED: {
    idle:
      'border-border bg-card text-muted-foreground hover:border-muted-foreground/50 hover:bg-muted hover:text-foreground',
    active:
      'border-border bg-muted text-foreground hover:border-muted-foreground/60 hover:bg-muted/80',
    countIdle: 'text-muted-foreground group-hover:text-foreground',
    countActive: 'text-muted-foreground',
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
    <ChipRow aria-label="Campaign status" className={className} role="group">
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
              'group',
              uiSurfaces.chip,
              adminKit.controlRadius,
              selected ? tone.active : tone.idle
            )}
            type="button"
            onClick={() => onChange(option.value)}
          >
            {option.label}
            <span
              className={cn(
                uiSurfaces.chipCount,
                selected ? tone.countActive : tone.countIdle
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
