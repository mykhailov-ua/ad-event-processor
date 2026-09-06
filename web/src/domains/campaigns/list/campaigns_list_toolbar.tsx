import { BarChart3, MoreHorizontal, Plus } from 'lucide-react';
import { useMemo } from 'react';
import { toast } from 'sonner';

import type { CampaignStatusTotals } from '@/api/campaigns_api';
import type { CustomerComboboxOption } from '@/shell/customer_combobox';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { ToolbarDateRangePicker } from '@/shell/toolbar_date_range_picker';
import { Input } from '@/components/ui/input';
import { CampaignListCountrySelect } from '@/domains/campaigns/list/campaign_list_country_select';
import type { CampaignsListFilterOption } from '@/domains/campaigns/list/campaigns_list_filter_select';
import { CampaignsListFilterSelect } from '@/domains/campaigns/list/campaigns_list_filter_select';
import type { CampaignListSummary } from '@/domains/campaigns/list/campaign_list_summary';
import { CampaignListSummaryBox } from '@/domains/campaigns/list/campaign_list_summary_box';
import { CampaignListStatusChips } from '@/domains/campaigns/list/campaign_list_status_chips';
import {
  campaignListArchiveButtonClass,
  campaignListFilterFieldClass,
  campaignListFilterLabelClass,
} from '@/domains/campaigns/list/campaign_list_classes';
import type {
  CampaignPacingFilter,
  CampaignStatusFilter,
} from '@/domains/campaigns/list/campaigns_list_types';
import { DirectoryFilterForm, FilterField, FilterPanel } from '@/shell/filter_panel';
import { PaginationPrevNext } from '@/shell/pagination_prev_next';
import { cn } from '@/lib/utils';

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
  listFacetsDegraded?: boolean;
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
  canGoPrev?: boolean;
  canGoNext?: boolean;
  paginationDisabled?: boolean;
  onPagePrev?: () => void;
  onPageNext?: () => void;
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
  listFacetsDegraded = false,
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
  canGoPrev = false,
  canGoNext = false,
  paginationDisabled = false,
  onPagePrev,
  onPageNext,
}: CampaignsListToolbarProps) {
  const bulkActionBusy = bulkBusy;
  const hasSelection = selectedCount > 0;
  const singleSelected = selectedCount === 1;
  const showWizardAction = onWizardClick != null;
  const showImportAction = onImportClick != null;

  function runBulkAction(allowed: boolean, hint: string, action?: () => void) {
    if (bulkActionBusy) {
      toast.message('Bulk action in progress');
      return;
    }
    if (!allowed) {
      toast.message(hint);
      return;
    }
    action?.();
  }

  const groupOptions = useMemo<CampaignsListFilterOption[]>(
    () => [
      { value: ALL_OPTION_VALUE, label: 'All groups' },
      ...customerOptions.map((customer) => ({
        value: customer.id,
        label: customer.name,
      })),
    ],
    [customerOptions]
  );

  const statusChipOptions = useMemo(
    () => [
      { value: '' as CampaignStatusFilter, label: 'All', count: statusTotals?.total },
      { value: 'ACTIVE' as CampaignStatusFilter, label: 'Active', count: statusTotals?.active },
      { value: 'PAUSED' as CampaignStatusFilter, label: 'Paused', count: statusTotals?.paused },
      {
        value: 'ARCHIVED' as CampaignStatusFilter,
        label: 'Archived',
        count: statusTotals?.archived,
      },
    ],
    [statusTotals]
  );

  return (
    <div className="flex w-full flex-col gap-3">
      <div className="flex flex-wrap items-center gap-4">
        <h1 className="m-0 shrink-0 text-lg font-normal text-foreground">Campaigns</h1>
        <div
          aria-label="Campaign actions"
          className="flex flex-wrap items-center gap-2"
          role="toolbar"
        >
          <Button type="button" variant="brand" onClick={onCreateClick}>
            <Plus className="h-3.5 w-3.5" aria-hidden />
            Quick create
          </Button>
          <div className="flex flex-nowrap items-center gap-2" aria-label="Selected campaigns">
            <Button
              type="button"
              variant="outline"
              title={singleSelected ? 'Clone selected campaign' : 'Select exactly one campaign'}
              onClick={() =>
                runBulkAction(singleSelected, 'Select exactly one campaign', onCloneClick)
              }
            >
              Clone
            </Button>
            <Button
              type="button"
              variant="outline"
              title={
                singleSelected ? 'Open report for selected campaign' : 'Select exactly one campaign'
              }
              onClick={() =>
                runBulkAction(singleSelected, 'Select exactly one campaign', onReportClick)
              }
            >
              <BarChart3 className="h-4 w-4" aria-hidden />
              Report
            </Button>
            <Button
              type="button"
              variant="outline"
              title={hasSelection ? 'Pause selected campaigns' : 'Select campaigns first'}
              onClick={() => runBulkAction(hasSelection, 'Select campaigns first', onPauseClick)}
            >
              Pause
            </Button>
            <Button
              type="button"
              variant="outline"
              title={hasSelection ? 'Resume selected campaigns' : 'Select campaigns first'}
              onClick={() => runBulkAction(hasSelection, 'Select campaigns first', onResumeClick)}
            >
              Resume
            </Button>
            <Button
              type="button"
              variant="outline"
              className={campaignListArchiveButtonClass}
              title={hasSelection ? 'Archive selected campaigns' : 'Select campaigns first'}
              onClick={() => runBulkAction(hasSelection, 'Select campaigns first', onArchiveClick)}
            >
              Archive
            </Button>
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                aria-label="More campaign actions"
                className="size-7 p-0"
                type="button"
                variant="outline"
              >
                <MoreHorizontal className="h-4 w-4" aria-hidden />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-44">
              {showWizardAction ? (
                <DropdownMenuItem onSelect={onWizardClick}>Guided setup</DropdownMenuItem>
              ) : null}
              {showImportAction ? (
                <DropdownMenuItem onSelect={onImportClick}>Import</DropdownMenuItem>
              ) : null}
              <DropdownMenuItem
                onSelect={() => {
                  if (fetching) {
                    toast.message('Refresh already in progress');
                    return;
                  }
                  onRefresh();
                }}
              >
                Refresh
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <div aria-label="Status and page summary" className="flex flex-wrap items-center gap-4">
        {statusTotals || statusTotalsLoading ? (
          <CampaignListStatusChips
            className="shrink-0"
            countsLoading={statusTotalsLoading}
            options={statusChipOptions}
            value={appliedStatus}
            onChange={onDraftStatusChange}
          />
        ) : null}
        <CampaignListSummaryBox
          filterTotalsCapped={filterTotalsCapped}
          filteredTotal={filteredTotal}
          metricsStale={metricsStale}
          summary={summary}
        />
      </div>

      <FilterPanel aria-label="List filters" className="bg-transparent p-0 pt-0" role="search">
        <DirectoryFilterForm
          layout="campaigns"
          onKeyDown={(event) => {
            if (event.key !== 'Enter' || event.defaultPrevented) {
              return;
            }
            const target = event.target;
            if (!(target instanceof HTMLInputElement)) {
              return;
            }
            if (target.id !== 'campaigns-budget-min' && target.id !== 'campaigns-budget-max') {
              return;
            }
            event.preventDefault();
            onBudgetFiltersApply();
          }}
        >
          <FilterField
            className={campaignListFilterFieldClass}
            label="Customer group"
            labelClassName={campaignListFilterLabelClass}
          >
            <CampaignsListFilterSelect
              aria-label="Customer group"
              options={groupOptions}
              value={draftCustomerId || ALL_OPTION_VALUE}
              onValueChange={(value) =>
                onDraftCustomerIdChange(value === ALL_OPTION_VALUE ? '' : value)
              }
            />
          </FilterField>

          <FilterField
            className={campaignListFilterFieldClass}
            label="Pacing"
            labelClassName={campaignListFilterLabelClass}
          >
            <CampaignsListFilterSelect
              aria-label="Pacing"
              options={PACING_FILTER_OPTIONS}
              value={draftPacing || ALL_OPTION_VALUE}
              onValueChange={(value) =>
                onDraftPacingChange(
                  value === ALL_OPTION_VALUE ? '' : (value as CampaignPacingFilter)
                )
              }
            />
          </FilterField>

          <FilterField
            className={campaignListFilterFieldClass}
            label="Owner"
            labelClassName={campaignListFilterLabelClass}
          >
            <CampaignsListFilterSelect
              aria-label="Owner"
              disabled={fetching || listFacetsFetching || listFacetsDegraded}
              options={ownerOptions}
              title={
                listFacetsDegraded
                  ? 'Owner filter requires list-facets API'
                  : 'Filter campaigns by owner'
              }
              value={draftOwnerUserId || ALL_OPTION_VALUE}
              onValueChange={(value) =>
                onDraftOwnerUserIdChange(value === ALL_OPTION_VALUE ? '' : value)
              }
            />
          </FilterField>

          <FilterField
            className={campaignListFilterFieldClass}
            label="Country"
            labelClassName={campaignListFilterLabelClass}
          >
            <CampaignListCountrySelect
              aria-label="Country"
              disabled={fetching || listFacetsFetching || listFacetsDegraded}
              options={countryOptions}
              title={
                listFacetsDegraded
                  ? 'Country filter requires list-facets API'
                  : 'Filter campaigns by target country'
              }
              value={draftCountry || ALL_OPTION_VALUE}
              onValueChange={(value) =>
                onDraftCountryChange(value === ALL_OPTION_VALUE ? '' : value)
              }
            />
          </FilterField>

          <FilterField
            className={campaignListFilterFieldClass}
            htmlFor="campaigns-budget-min"
            label="Budget min ($)"
            labelClassName={campaignListFilterLabelClass}
          >
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

          <FilterField
            className={campaignListFilterFieldClass}
            htmlFor="campaigns-budget-max"
            label="Budget max ($)"
            labelClassName={campaignListFilterLabelClass}
          >
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

          <ToolbarDateRangePicker
            className={cn(campaignListFilterFieldClass, 'min-w-0')}
            disabled={fetching}
            from={draftStatsFrom}
            id="campaign-list-stats-range"
            label="Period"
            labelClassName={campaignListFilterLabelClass}
            to={draftStatsTo}
            onChange={onStatsRangeChange}
          />

          {onPagePrev && onPageNext ? (
            <FilterField
              className="min-w-[10rem]"
              label="Page"
              labelClassName={campaignListFilterLabelClass}
            >
              <PaginationPrevNext
                canGoNext={canGoNext}
                canGoPrev={canGoPrev}
                className="w-full"
                disabled={paginationDisabled}
                layout="split"
                nextLabel="Next"
                prevLabel="Prev"
                variant="outline"
                onNext={onPageNext}
                onPrev={onPagePrev}
              />
            </FilterField>
          ) : null}
        </DirectoryFilterForm>
      </FilterPanel>
    </div>
  );
}
