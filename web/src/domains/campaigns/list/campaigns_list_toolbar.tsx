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
import { DateRangePicker } from '@/components/ui/date_range_picker';
import { Input } from '@/components/ui/input';
import { CampaignListCountrySelect } from '@/domains/campaigns/list/campaign_list_country_select';
import type { CampaignsListFilterOption } from '@/domains/campaigns/list/campaigns_list_filter_select';
import { CampaignsListFilterSelect } from '@/domains/campaigns/list/campaigns_list_filter_select';
import type { CampaignListSummary } from '@/domains/campaigns/list/campaign_list_summary';
import { CampaignListSummaryBox } from '@/domains/campaigns/list/campaign_list_summary_box';
import { CampaignListStatusChips } from '@/domains/campaigns/list/campaign_list_status_chips';
import type { CampaignPacingFilter, CampaignStatusFilter } from '@/domains/campaigns/list/campaigns_list_types';
import {
  DirectoryFilterForm,
  FilterField,
  FilterPanel,
} from '@/shell/filter_panel';

const ALL_OPTION_VALUE = '__all__';
const FILTER_LABEL_CLASS = 'admin-campaigns-filter-label';

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
}: CampaignsListToolbarProps) {
  const bulkActionBusy = bulkBusy;
  const hasSelection = selectedCount > 0;
  const singleSelected = selectedCount === 1;

  function runBulkAction(
    allowed: boolean,
    hint: string,
    action?: () => void,
  ) {
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
    <div className="admin-campaigns-toolbar">
      <div className="admin-campaigns-toolbar__page-header">
        <h1 className="admin-campaigns-toolbar__title">Campaigns</h1>
        <div aria-label="Campaign actions" className="admin-campaigns-toolbar__actions" role="toolbar">
          <Button className="admin-campaigns-toolbar__create-btn" type="button" variant="brand" onClick={onCreateClick}>
            <Plus className="h-3.5 w-3.5" aria-hidden />
            Create
          </Button>
          <div className="flex flex-nowrap items-center gap-2" aria-label="Selected campaigns">
          <Button
            type="button"
            variant="outline"
            className="admin-campaigns-toolbar__outline-btn"
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
            className="admin-campaigns-toolbar__outline-btn"
            title={singleSelected ? 'Open report for selected campaign' : 'Select exactly one campaign'}
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
            className="admin-campaigns-toolbar__outline-btn"
            title={hasSelection ? 'Pause selected campaigns' : 'Select campaigns first'}
            onClick={() => runBulkAction(hasSelection, 'Select campaigns first', onPauseClick)}
          >
            Pause
          </Button>
          <Button
            type="button"
            variant="outline"
            className="admin-campaigns-toolbar__outline-btn"
            title={hasSelection ? 'Resume selected campaigns' : 'Select campaigns first'}
            onClick={() => runBulkAction(hasSelection, 'Select campaigns first', onResumeClick)}
          >
            Resume
          </Button>
          <Button
            type="button"
            variant="outline"
            className="admin-campaigns-toolbar__archive-btn"
            title={hasSelection ? 'Archive selected campaigns' : 'Select campaigns first'}
            onClick={() => runBulkAction(hasSelection, 'Select campaigns first', onArchiveClick)}
          >
            Archive
          </Button>
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                type="button"
                variant="outline"
                className="admin-campaigns-toolbar__outline-btn"
                size="icon"
                aria-label="More campaign actions"
              >
                <MoreHorizontal className="h-4 w-4" aria-hidden />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-44">
              <DropdownMenuItem
                onSelect={() => {
                  if (!onWizardClick) {
                    toast.message('Wizard is not available');
                    return;
                  }
                  onWizardClick();
                }}
              >
                Wizard
              </DropdownMenuItem>
              <DropdownMenuItem
                onSelect={() => {
                  if (!onImportClick) {
                    toast.message('Import is not available');
                    return;
                  }
                  onImportClick();
                }}
              >
                Import
              </DropdownMenuItem>
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

      <div aria-label="Status and page summary" className="admin-campaigns-toolbar__status-row">
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

      <FilterPanel aria-label="List filters" className="admin-campaigns-toolbar__filters !bg-transparent !p-0" role="search">
        <DirectoryFilterForm layout="campaigns" onSubmit={(event) => event.preventDefault()}>
          <FilterField className="admin-campaigns-filter-field" label="Customer group" labelClassName={FILTER_LABEL_CLASS}>
            <CampaignsListFilterSelect
              aria-label="Customer group"
              options={groupOptions}
              value={draftCustomerId || ALL_OPTION_VALUE}
              onValueChange={(value) =>
                onDraftCustomerIdChange(value === ALL_OPTION_VALUE ? '' : value)
              }
            />
          </FilterField>

          <FilterField className="admin-campaigns-filter-field" label="Pacing" labelClassName={FILTER_LABEL_CLASS}>
            <CampaignsListFilterSelect
              aria-label="Pacing"
              options={PACING_FILTER_OPTIONS}
              value={draftPacing || ALL_OPTION_VALUE}
              onValueChange={(value) =>
                onDraftPacingChange(value === ALL_OPTION_VALUE ? '' : (value as CampaignPacingFilter))
              }
            />
          </FilterField>

          <FilterField className="admin-campaigns-filter-field" label="Owner" labelClassName={FILTER_LABEL_CLASS}>
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

          <FilterField className="admin-campaigns-filter-field" label="Country" labelClassName={FILTER_LABEL_CLASS}>
            <CampaignListCountrySelect
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

          <FilterField className="admin-campaigns-filter-field" htmlFor="campaigns-budget-min" label="Budget min ($)" labelClassName={FILTER_LABEL_CLASS}>
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

          <FilterField className="admin-campaigns-filter-field" htmlFor="campaigns-budget-max" label="Budget max ($)" labelClassName={FILTER_LABEL_CLASS}>
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
            className="admin-campaigns-filter-field min-w-0"
            disabled={fetching}
            from={draftStatsFrom}
            id="campaign-list-stats-range"
            label="Period"
            labelClassName={FILTER_LABEL_CLASS}
            to={draftStatsTo}
            variant="campaigns"
            onChange={onStatsRangeChange}
          />
        </DirectoryFilterForm>
      </FilterPanel>
    </div>
  );
}
