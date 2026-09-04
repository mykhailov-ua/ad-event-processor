import { MoreHorizontal } from 'lucide-react';
import { useMemo } from 'react';

import type { CampaignStatusTotals } from '@/api/campaigns_api';
import type { CustomerComboboxOption } from '@/shell/customer_combobox';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { DateRangePicker } from '@/components/ui/date_range_picker';
import { Input } from '@/components/ui/input';
import {
  CampaignsListFilterSelect,
  type CampaignsListFilterOption,
} from '@/domains/campaigns/list/campaigns_list_filter_select';
import {
  formatCampaignListSummaryLine,
  type CampaignListSummary,
} from '@/domains/campaigns/list/campaign_list_summary';
import type { CampaignListColumnPrefs } from '@/domains/campaigns/list/campaign_list_columns';
import { CAMPAIGN_LIST_FILTER_TOTALS_MAX } from '@/domains/campaigns/list/campaign_list_limits';
import { CampaignListColumnsMenu } from '@/domains/campaigns/list/campaign_list_columns_menu';
import type { CampaignPacingFilter, CampaignStatusFilter } from '@/domains/campaigns/list/campaigns_list_types';
import {
  DirectoryFilterForm,
  FilterField,
  FilterPanel,
} from '@/shell/filter_panel';
import { ToggleChipGroup } from '@/shell/toggle_chip_group';

const ALL_OPTION_VALUE = '__all__';

const PACING_FILTER_OPTIONS: CampaignsListFilterOption[] = [
  { value: ALL_OPTION_VALUE, label: 'All pacing' },
  { value: 'EVEN', label: 'Even' },
  { value: 'ASAP', label: 'ASAP' },
];

export type CampaignsListToolbarProps = {
  draftCustomerId: string;
  appliedStatus: CampaignStatusFilter;
  draftPacing: CampaignPacingFilter;
  draftBudgetMinUsd: string;
  draftBudgetMaxUsd: string;
  draftOwnerUserId: string;
  draftCountry: string;
  draftStatsFrom: string;
  draftStatsTo: string;
  customerOptions: CustomerComboboxOption[];
  ownerOptions: CampaignsListFilterOption[];
  countryOptions: CampaignsListFilterOption[];
  listFacetsFetching?: boolean;
  filterTotalsCapped?: boolean;
  filteredTotal?: number;
  metricsStale?: boolean;
  summary: CampaignListSummary;
  statusTotals?: CampaignStatusTotals;
  statusTotalsLoading?: boolean;
  selectedCount?: number;
  bulkBusy?: boolean;
  fetching?: boolean;
  onDraftCustomerIdChange: (customerId: string) => void;
  onDraftStatusChange: (status: CampaignStatusFilter) => void;
  onDraftPacingChange: (pacing: CampaignPacingFilter) => void;
  onDraftBudgetMinUsdChange: (value: string) => void;
  onDraftBudgetMaxUsdChange: (value: string) => void;
  onDraftOwnerUserIdChange: (userId: string) => void;
  onDraftCountryChange: (country: string) => void;
  onStatsRangeChange: (from: string, to: string) => void;
  onBudgetFiltersApply: () => void;
  onRefresh: () => void;
  onCreateClick: () => void;
  onWizardClick?: () => void;
  onImportClick?: () => void;
  onCloneClick?: () => void;
  onReportClick?: () => void;
  onPauseClick?: () => void;
  onResumeClick?: () => void;
  onArchiveClick?: () => void;
  columnPrefs: CampaignListColumnPrefs;
  onColumnPrefsChange: (prefs: CampaignListColumnPrefs) => void;
  onResetWorkspaceClick: () => void;
};

export function CampaignsListToolbar({
  draftCustomerId,
  appliedStatus,
  draftPacing,
  draftBudgetMinUsd,
  draftBudgetMaxUsd,
  draftOwnerUserId,
  draftCountry,
  draftStatsFrom,
  draftStatsTo,
  customerOptions,
  ownerOptions,
  countryOptions,
  listFacetsFetching = false,
  filterTotalsCapped = false,
  filteredTotal = 0,
  metricsStale = false,
  summary,
  statusTotals,
  statusTotalsLoading = false,
  selectedCount = 0,
  bulkBusy = false,
  fetching = false,
  onDraftCustomerIdChange,
  onDraftStatusChange,
  onDraftPacingChange,
  onDraftBudgetMinUsdChange,
  onDraftBudgetMaxUsdChange,
  onDraftOwnerUserIdChange,
  onDraftCountryChange,
  onStatsRangeChange,
  onBudgetFiltersApply,
  onRefresh,
  onCreateClick,
  onWizardClick,
  onImportClick,
  onCloneClick,
  onReportClick,
  onPauseClick,
  onResumeClick,
  onArchiveClick,
  columnPrefs,
  onColumnPrefsChange,
  onResetWorkspaceClick,
}: CampaignsListToolbarProps) {
  const bulkActionBusy = bulkBusy;
  const hasSelection = selectedCount > 0;
  const singleSelected = selectedCount === 1;

  const groupOptions = useMemo<CampaignsListFilterOption[]>(
    () => [
      { value: ALL_OPTION_VALUE, label: 'All groups' },
      ...customerOptions.map((customer) => ({
        value: customer.id,
        label: customer.name,
      })),
    ],
    [customerOptions],
  );

  const statusChipOptions = useMemo(
    () => [
      { value: '' as CampaignStatusFilter, label: 'All', count: statusTotals?.total },
      { value: 'ACTIVE' as CampaignStatusFilter, label: 'Active', count: statusTotals?.active },
      { value: 'PAUSED' as CampaignStatusFilter, label: 'Paused', count: statusTotals?.paused },
      { value: 'ARCHIVED' as CampaignStatusFilter, label: 'Archived', count: statusTotals?.archived },
    ],
    [statusTotals],
  );

  return (
    <div className="flex w-full flex-col gap-2">
      <div aria-label="Campaign actions" className="flex flex-wrap items-center gap-2" role="toolbar">
        <Button type="button" onClick={onCreateClick}>
          Create
        </Button>
        <div className="flex flex-wrap items-center gap-1" aria-label="Selected campaigns">
          <Button
            type="button"
            variant="secondary"
            disabled={bulkActionBusy || !singleSelected}
            title={singleSelected ? 'Clone selected campaign' : 'Select exactly one campaign'}
            onClick={() => onCloneClick?.()}
          >
            Clone
          </Button>
          <Button
            type="button"
            variant="secondary"
            disabled={bulkActionBusy || !singleSelected}
            title={singleSelected ? 'Open report for selected campaign' : 'Select exactly one campaign'}
            onClick={() => onReportClick?.()}
          >
            Report
          </Button>
          <Button
            type="button"
            variant="secondary"
            disabled={bulkActionBusy || !hasSelection}
            title={hasSelection ? 'Pause selected campaigns' : 'Select campaigns first'}
            onClick={() => onPauseClick?.()}
          >
            Pause
          </Button>
          <Button
            type="button"
            variant="secondary"
            disabled={bulkActionBusy || !hasSelection}
            title={hasSelection ? 'Resume selected campaigns' : 'Select campaigns first'}
            onClick={() => onResumeClick?.()}
          >
            Resume
          </Button>
          <Button
            type="button"
            variant="destructive"
            disabled={bulkActionBusy || !hasSelection}
            title={hasSelection ? 'Archive selected campaigns' : 'Select campaigns first'}
            onClick={() => onArchiveClick?.()}
          >
            Archive
          </Button>
        </div>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              aria-label="More campaign actions"
              disabled={fetching}
            >
              <MoreHorizontal className="h-4 w-4" aria-hidden />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-44">
            <DropdownMenuItem disabled={!onWizardClick} onSelect={() => onWizardClick?.()}>
              Wizard
            </DropdownMenuItem>
            <DropdownMenuItem disabled={!onImportClick} onSelect={() => onImportClick?.()}>
              Import
            </DropdownMenuItem>
            <DropdownMenuItem disabled={fetching} onSelect={() => onRefresh()}>
              Refresh
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <div
        aria-label="Status and page summary"
        className="flex flex-wrap items-center gap-3 border-t border-zinc-200 pt-2 dark:border-zinc-800"
      >
        {statusTotals || statusTotalsLoading ? (
          <ToggleChipGroup
            countsLoading={statusTotalsLoading}
            options={statusChipOptions}
            value={appliedStatus}
            onChange={onDraftStatusChange}
          />
        ) : null}
        <div className="flex flex-wrap items-center gap-2 text-sm text-zinc-500 dark:text-zinc-400">
          <span>{formatCampaignListSummaryLine(summary)}</span>
          {summary.staleCount > 0 ? (
            <span className="text-xs">
              {summary.scope === 'filter' ? 'Filtered totals may be stale' : `Stale stats: ${summary.staleCount}`}
            </span>
          ) : null}
          {summary.marginBreachCount > 0 ? (
            <span className="text-xs">Margin breach: {summary.marginBreachCount}</span>
          ) : null}
          {filterTotalsCapped ? (
            <span className="text-xs">
              Filter totals unavailable above {CAMPAIGN_LIST_FILTER_TOTALS_MAX.toLocaleString()} campaigns (
              {filteredTotal.toLocaleString()} matched)
            </span>
          ) : null}
          {metricsStale && !filterTotalsCapped ? (
            <span className="text-xs">Page metrics may be stale</span>
          ) : null}
        </div>
      </div>

      <FilterPanel aria-label="List filters" role="search">
        <DirectoryFilterForm layout="directory" onSubmit={(event) => event.preventDefault()}>
          <FilterField label="Customer group">
            <CampaignsListFilterSelect
              aria-label="Customer group"
              options={groupOptions}
              value={draftCustomerId || ALL_OPTION_VALUE}
              onValueChange={(value) =>
                onDraftCustomerIdChange(value === ALL_OPTION_VALUE ? '' : value)
              }
            />
          </FilterField>

          <FilterField label="Pacing">
            <CampaignsListFilterSelect
              aria-label="Pacing"
              options={PACING_FILTER_OPTIONS}
              value={draftPacing || ALL_OPTION_VALUE}
              onValueChange={(value) =>
                onDraftPacingChange(value === ALL_OPTION_VALUE ? '' : (value as CampaignPacingFilter))
              }
            />
          </FilterField>

          <FilterField label="Owner">
            <CampaignsListFilterSelect
              aria-label="Owner"
              disabled={fetching || listFacetsFetching}
              options={ownerOptions}
              title="Filter campaigns by owner"
              value={draftOwnerUserId || ALL_OPTION_VALUE}
              onValueChange={(value) =>
                onDraftOwnerUserIdChange(value === ALL_OPTION_VALUE ? '' : value)
              }
            />
          </FilterField>

          <FilterField label="Country">
            <CampaignsListFilterSelect
              aria-label="Country"
              disabled={fetching || listFacetsFetching}
              options={countryOptions}
              title="Filter campaigns by target country"
              value={draftCountry || ALL_OPTION_VALUE}
              onValueChange={(value) =>
                onDraftCountryChange(value === ALL_OPTION_VALUE ? '' : value)
              }
            />
          </FilterField>

          <FilterField htmlFor="campaigns-budget-min" label="Budget min ($)">
            <Input
              id="campaigns-budget-min"
              disabled={fetching}
              inputMode="decimal"
              placeholder="0.00"
              title="Server filter; applied when field loses focus"
              value={draftBudgetMinUsd}
              onBlur={onBudgetFiltersApply}
              onChange={(event) => onDraftBudgetMinUsdChange(event.target.value)}
            />
          </FilterField>

          <FilterField htmlFor="campaigns-budget-max" label="Budget max ($)">
            <Input
              id="campaigns-budget-max"
              disabled={fetching}
              inputMode="decimal"
              placeholder="0.00"
              title="Server filter; applied when field loses focus"
              value={draftBudgetMaxUsd}
              onBlur={onBudgetFiltersApply}
              onChange={(event) => onDraftBudgetMaxUsdChange(event.target.value)}
            />
          </FilterField>

          <DateRangePicker
            className="min-w-0"
            disabled={fetching}
            from={draftStatsFrom}
            id="campaign-list-stats-range"
            label="Period"
            to={draftStatsTo}
            onChange={onStatsRangeChange}
          />

          <div className="flex flex-wrap items-end gap-2">
            <CampaignListColumnsMenu
              columnPrefs={columnPrefs}
              disabled={fetching}
              onColumnPrefsChange={onColumnPrefsChange}
            />
            <Button
              type="button"
              variant="ghost"
              disabled={fetching}
              title="Reset columns and widths"
              onClick={onResetWorkspaceClick}
            >
              Reset view
            </Button>
          </div>
        </DirectoryFilterForm>
      </FilterPanel>
    </div>
  );
}
