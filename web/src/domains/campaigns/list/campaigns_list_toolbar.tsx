import { useMemo, type ReactNode } from 'react';
import { MoreHorizontal, Plus } from 'lucide-react';
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
import {
  CampaignsListFilterSelect,
  CampaignsListSearchableFilterSelect,
} from '@/domains/campaigns/list/campaigns_list_filter_select';
import { ListRefreshBand } from '@/shell/list_refresh_band';
import { CampaignListStatusChips } from '@/domains/campaigns/list/campaign_list_status_chips';
import type {
  CampaignPacingFilter,
  CampaignStatusFilter,
} from '@/domains/campaigns/list/campaigns_list_types';
import {
  CAMPAIGNS_FILTER_ROW,
  FilterField,
  FilterPanel,
} from '@/shell/filter_panel';
import { adminTypography } from '@/lib/admin_kit';
import {
  StatusMetricsBand,
  ToolbarBandActions,
  DirectoryStack,
} from '@/shell/ui_bands';

const ALL_OPTION_VALUE = '__all__';

const PACING_FILTER_OPTIONS: CampaignsListFilterOption[] = [
  { value: ALL_OPTION_VALUE, label: 'All pacing' },
  { value: 'EVEN', label: 'Even' },
  { value: 'ASAP', label: 'ASAP' },
];

export type CampaignsListToolbarProps = {
  draftCustomerId: string;
  draftStatus: CampaignStatusFilter;
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
  listLastUpdatedAt?: string | null;
  listRevalidating?: boolean;
  statusTotals?: CampaignStatusTotals;
  statusTotalsLoading?: boolean;
  fetching?: boolean;
  onDraftCustomerIdChange: (customerId: string) => void;
  onDraftStatusChange: (status: CampaignStatusFilter) => void;
  onDraftPacingChange: (pacing: CampaignPacingFilter) => void;
  onDraftBudgetMinUsdChange: (value: string) => void;
  onDraftBudgetMaxUsdChange: (value: string) => void;
  onDraftOwnerUserIdChange: (userId: string) => void;
  onDraftCountryChange: (country: string) => void;
  onStatsRangeChange: (from: string, to: string) => void;
  onDirectoryFiltersApply: () => void;
  onRefresh: () => void;
  onCreateClick: () => void;
  onWizardClick?: () => void;
  onImportClick?: () => void;
  overlaysBusy?: boolean;
  /** Rendered beside Apply inside the filter panel (e.g. row selection actions). */
  filterFooter?: ReactNode;
};

export function CampaignsListToolbar({
  draftCustomerId,
  draftStatus,
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
  listLastUpdatedAt = null,
  listRevalidating = false,
  statusTotals,
  statusTotalsLoading = false,
  fetching = false,
  onDraftCustomerIdChange,
  onDraftStatusChange,
  onDraftPacingChange,
  onDraftBudgetMinUsdChange,
  onDraftBudgetMaxUsdChange,
  onDraftOwnerUserIdChange,
  onDraftCountryChange,
  onStatsRangeChange,
  onDirectoryFiltersApply,
  onRefresh,
  onCreateClick,
  onWizardClick,
  onImportClick,
  overlaysBusy = false,
  filterFooter,
}: CampaignsListToolbarProps) {
  const showWizardAction = onWizardClick != null;
  const showImportAction = onImportClick != null;

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
      {
        value: 'WARNINGS' as CampaignStatusFilter,
        label: 'Warnings',
        count: statusTotals?.warnings,
      },
    ],
    [statusTotals]
  );

  return (
    <DirectoryStack className="gap-2">
      <FilterPanel aria-label="List filters" className="gap-2 p-3" role="search">
        <div className="flex flex-wrap items-center gap-2">
          <ToolbarBandActions aria-label="Campaign actions">
            <Button type="button" variant="brand" onClick={onCreateClick}>
              <Plus aria-hidden />
              Quick create
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  aria-label="More campaign actions"
                  type="button"
                  variant="outline"
                >
                  <MoreHorizontal aria-hidden />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                {showWizardAction ? (
                  <DropdownMenuItem onSelect={onWizardClick}>Guided setup</DropdownMenuItem>
                ) : null}
                {showImportAction ? (
                  <DropdownMenuItem onSelect={onImportClick}>Import</DropdownMenuItem>
                ) : null}
              </DropdownMenuContent>
            </DropdownMenu>
          </ToolbarBandActions>
          {overlaysBusy ? (
            <p className={adminTypography.bodyMuted}>
              Finish or close the open campaign dialog before starting another create flow.
            </p>
          ) : null}
          <div className="min-w-0 flex-1">
            <StatusMetricsBand aria-label="Status filters">
              {statusTotals || statusTotalsLoading ? (
                <CampaignListStatusChips
                  countsLoading={statusTotalsLoading}
                  options={statusChipOptions}
                  value={draftStatus}
                  onChange={onDraftStatusChange}
                />
              ) : null}
            </StatusMetricsBand>
          </div>
          <ListRefreshBand
            ariaLabel="Refresh campaign list"
            lastUpdatedAt={listLastUpdatedAt}
            loading={fetching || listRevalidating}
            onRefresh={onRefresh}
          />
        </div>

        <form
          className="grid w-full gap-2"
          onSubmit={(event) => {
            event.preventDefault();
            onDirectoryFiltersApply();
          }}
        >
          <div className={CAMPAIGNS_FILTER_ROW}>
          <FilterField compact label="Customer group">
            <CampaignsListSearchableFilterSelect
              aria-label="Customer group"
              options={groupOptions}
              searchPlaceholder="All groups"
              value={draftCustomerId || ALL_OPTION_VALUE}
              onValueChange={(value) =>
                onDraftCustomerIdChange(value === ALL_OPTION_VALUE ? '' : value)
              }
            />
          </FilterField>

          <FilterField compact label="Pacing">
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

          <FilterField compact label="Owner">
            <CampaignsListSearchableFilterSelect
              aria-label="Owner"
              disabled={fetching || listFacetsFetching || listFacetsDegraded}
              options={ownerOptions}
              searchPlaceholder="All owners"
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

          <FilterField compact label="Country">
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

          <FilterField compact htmlFor="campaigns-budget-min" label="Budget min ($)">
            <Input
              id="campaigns-budget-min"
              disabled={fetching}
              inputMode="decimal"
              placeholder="Min budget"
              value={draftBudgetMinUsd}
              onChange={(event) => onDraftBudgetMinUsdChange(event.target.value)}
            />
          </FilterField>

          <FilterField compact htmlFor="campaigns-budget-max" label="Budget max ($)">
            <Input
              id="campaigns-budget-max"
              disabled={fetching}
              inputMode="decimal"
              placeholder="Max budget"
              value={draftBudgetMaxUsd}
              onChange={(event) => onDraftBudgetMaxUsdChange(event.target.value)}
            />
          </FilterField>

          <ToolbarDateRangePicker
            disabled={fetching}
            from={draftStatsFrom}
            id="campaign-list-stats-range"
            label="Period"
            to={draftStatsTo}
            onChange={onStatsRangeChange}
          />

          <FilterField compact hideLabel label="Apply filters">
            <div className="flex w-fit max-w-full flex-wrap items-center gap-2">
              <Button disabled={fetching} shape="default" type="submit" variant="brand">
                Apply
              </Button>
              {filterFooter}
            </div>
          </FilterField>

            </div>
        </form>
      </FilterPanel>
    </DirectoryStack>
  );
}
