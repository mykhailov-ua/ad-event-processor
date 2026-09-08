import { useMemo } from 'react';
import { Plus } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { ListRefreshBand } from '@/shell/list_refresh_band';
import {
  buildLandersHostingChipOptions,
  LandersListHostingChips,
} from '@/domains/creative/landers_list_hosting_chips';
import { LandersListSummaryBox } from '@/domains/creative/landers_list_summary_box';
import type { LanderHostingFilter } from '@/domains/creative/landers_list_types';
import type { LandersHostingCounts } from '@/domains/creative/use_landers_list_view';
import { CreativeDirectoryStack } from '@/domains/creative/creative_directory_stack';
import { campaignListFilterFieldClass } from '@/domains/campaigns/list/campaign_list_classes';
import { FilterField, FilterPanel, FILTER_PANEL_FLAT_CLASS } from '@/shell/filter_panel';
import { StatusMetricsBand, ToolbarBand, ToolbarBandActions } from '@/shell/ui_bands';
import { cn } from '@/lib/utils';

export type LandersListToolbarProps = {
  draftSearch: string;
  hostingFilter: LanderHostingFilter;
  hostingCounts: LandersHostingCounts;
  filteredTotal: number;
  filtersActive: boolean;
  fetching?: boolean;
  listLastUpdatedAt?: string | null;
  listRevalidating?: boolean;
  acting?: boolean;
  onDraftSearchChange: (value: string) => void;
  onHostingFilterChange: (value: LanderHostingFilter) => void;
  onRefresh: () => void;
  onCreateClick: () => void;
};

export function LandersListToolbar({
  draftSearch,
  hostingFilter,
  hostingCounts,
  filteredTotal,
  filtersActive,
  fetching = false,
  listLastUpdatedAt = null,
  listRevalidating = false,
  acting = false,
  onDraftSearchChange,
  onHostingFilterChange,
  onRefresh,
  onCreateClick,
}: LandersListToolbarProps) {
  const chipOptions = useMemo(() => buildLandersHostingChipOptions(hostingCounts), [hostingCounts]);

  return (
    <CreativeDirectoryStack>
      <ToolbarBand split>
        <ToolbarBandActions aria-label="Lander actions">
          <Button disabled={acting} type="button" variant="brand" onClick={onCreateClick}>
            <Plus className="h-3.5 w-3.5" aria-hidden />
            Create lander
          </Button>
        </ToolbarBandActions>
        <ListRefreshBand
          ariaLabel="Refresh lander list"
          disabled={acting}
          lastUpdatedAt={listLastUpdatedAt}
          loading={fetching || listRevalidating}
          title="Refresh lander list"
          onRefresh={onRefresh}
        />
      </ToolbarBand>

      <StatusMetricsBand aria-label="Hosting summary">
        <LandersListHostingChips
          className="shrink-0"
          options={chipOptions}
          value={hostingFilter}
          onChange={onHostingFilterChange}
        />
        <LandersListSummaryBox
          filteredTotal={filteredTotal}
          filtersActive={filtersActive}
          total={hostingCounts.total}
        />
      </StatusMetricsBand>

      <FilterPanel
        aria-label="List filters"
        className={cn(FILTER_PANEL_FLAT_CLASS, 'w-full')}
        role="search"
      >
        <FilterField className={cn(campaignListFilterFieldClass, 'max-w-xl')} label="Search">
          <Input
            aria-label="Search landers"
            className="w-full"
            placeholder="Name, URL, or ID..."
            value={draftSearch}
            onChange={(event) => onDraftSearchChange(event.target.value)}
          />
        </FilterField>
      </FilterPanel>
    </CreativeDirectoryStack>
  );
}
