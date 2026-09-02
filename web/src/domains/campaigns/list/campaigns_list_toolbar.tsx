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
import {
  CampaignsListFilterSelect,
  type CampaignsListFilterOption,
} from '@/domains/campaigns/list/campaigns_list_filter_select';
import {
  formatCampaignListSummaryLine,
  type CampaignListSummary,
} from '@/domains/campaigns/list/campaign_list_summary';
import type { CampaignListColumnPrefs } from '@/domains/campaigns/list/campaign_list_columns';
import { CampaignListColumnsMenu } from '@/domains/campaigns/list/campaign_list_columns_menu';
import type { CampaignListRowDisplayPrefs } from '@/domains/campaigns/list/campaign_list_row_display';
import type { CampaignPacingFilter, CampaignStatusFilter } from '@/domains/campaigns/list/campaigns_list_types';
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
  rowDisplayPrefs: CampaignListRowDisplayPrefs;
  onRowDisplayPrefsChange: (prefs: CampaignListRowDisplayPrefs) => void;
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
  rowDisplayPrefs,
  onRowDisplayPrefsChange,
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

  return (
    <div className="admin-campaigns-toolbar">
      <div aria-label="Campaign actions" className="admin-toolbar-row admin-campaigns-toolbar__actions" role="toolbar">
        <div className="admin-toolbar-group" aria-label="Create">
          <Button type="button" onClick={onCreateClick}>
            Create
          </Button>
        </div>
        <div className="admin-toolbar-group" aria-label="Selected campaigns">
          <Button
            type="button"
            variant="outline"
            disabled={bulkActionBusy || !singleSelected}
            title={singleSelected ? 'Clone selected campaign' : 'Select exactly one campaign'}
            onClick={() => onCloneClick?.()}
          >
            Clone
          </Button>
          <Button
            type="button"
            variant="outline"
            disabled={bulkActionBusy || !singleSelected}
            title={singleSelected ? 'Open report for selected campaign' : 'Select exactly one campaign'}
            onClick={() => onReportClick?.()}
          >
            Report
          </Button>
          <Button
            type="button"
            variant="outline"
            disabled={bulkActionBusy || !hasSelection}
            title={hasSelection ? 'Pause selected campaigns' : 'Select campaigns first'}
            onClick={() => onPauseClick?.()}
          >
            Pause
          </Button>
          <Button
            type="button"
            variant="outline"
            disabled={bulkActionBusy || !hasSelection}
            title={hasSelection ? 'Resume selected campaigns' : 'Select campaigns first'}
            onClick={() => onResumeClick?.()}
          >
            Resume
          </Button>
          <Button
            type="button"
            variant="outline"
            disabled={bulkActionBusy || !hasSelection}
            title={hasSelection ? 'Archive selected campaigns' : 'Select campaigns first'}
            onClick={() => onArchiveClick?.()}
          >
            Archive
          </Button>
        </div>
        <div className="admin-toolbar-group" aria-label="More actions">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                type="button"
                variant="outline"
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
      </div>

      <div aria-label="Status and page summary" className="admin-toolbar-summary admin-campaigns-toolbar__summary">
        {statusTotalsLoading ? (
          <span className="admin-muted">Loading counts...</span>
        ) : statusTotals ? (
          <span className="admin-status-links" role="group">
            <button
              aria-current={appliedStatus === '' ? 'true' : undefined}
              className={cn('admin-text-link', appliedStatus === '' && 'is-active')}
              type="button"
              onClick={() => onDraftStatusChange('')}
            >
              All {statusTotals.total}
            </button>
            <span aria-hidden className="admin-summary-sep">
               / 
            </span>
            <button
              aria-current={appliedStatus === 'ACTIVE' ? 'true' : undefined}
              className={cn('admin-text-link', appliedStatus === 'ACTIVE' && 'is-active')}
              type="button"
              onClick={() => onDraftStatusChange('ACTIVE')}
            >
              Active {statusTotals.active}
            </button>
            <span aria-hidden className="admin-summary-sep">
               / 
            </span>
            <button
              aria-current={appliedStatus === 'PAUSED' ? 'true' : undefined}
              className={cn('admin-text-link', appliedStatus === 'PAUSED' && 'is-active')}
              type="button"
              onClick={() => onDraftStatusChange('PAUSED')}
            >
              Paused {statusTotals.paused}
            </button>
            <span aria-hidden className="admin-summary-sep">
               / 
            </span>
            <button
              aria-current={appliedStatus === 'ARCHIVED' ? 'true' : undefined}
              className={cn('admin-text-link', appliedStatus === 'ARCHIVED' && 'is-active')}
              type="button"
              onClick={() => onDraftStatusChange('ARCHIVED')}
            >
              Archived {statusTotals.archived}
            </button>
          </span>
        ) : null}
        {statusTotals && !statusTotalsLoading ? (
          <span aria-hidden className="admin-summary-sep">
            |
          </span>
        ) : null}
        <span className="admin-muted">{formatCampaignListSummaryLine(summary)}</span>
        {summary.staleCount > 0 ? (
          <span className="admin-stat-note">Stale stats: {summary.staleCount}</span>
        ) : null}
        {summary.marginBreachCount > 0 ? (
          <span className="admin-stat-note">Margin breach: {summary.marginBreachCount}</span>
        ) : null}
      </div>

      <div
        aria-label="List filters"
        className="admin-toolbar-row admin-toolbar-row--filters admin-campaigns-toolbar__filters"
        role="search"
      >
        <CampaignsListFilterSelect
          aria-label="Customer group"
          options={groupOptions}
          value={draftCustomerId || ALL_OPTION_VALUE}
          onValueChange={(value) =>
            onDraftCustomerIdChange(value === ALL_OPTION_VALUE ? '' : value)
          }
        />

        <CampaignsListFilterSelect
          aria-label="Pacing"
          options={PACING_FILTER_OPTIONS}
          value={draftPacing || ALL_OPTION_VALUE}
          onValueChange={(value) =>
            onDraftPacingChange(value === ALL_OPTION_VALUE ? '' : (value as CampaignPacingFilter))
          }
        />

        <CampaignsListFilterSelect
          aria-label="Owner"
          disabled={fetching || ownerOptions.length <= 1}
          options={ownerOptions}
          title="Filter campaigns by owner"
          value={draftOwnerUserId || ALL_OPTION_VALUE}
          onValueChange={(value) =>
            onDraftOwnerUserIdChange(value === ALL_OPTION_VALUE ? '' : value)
          }
        />

        <CampaignsListFilterSelect
          aria-label="Country (current page)"
          options={countryOptions}
          title="Client-side filter on loaded rows"
          value={draftCountry || ALL_OPTION_VALUE}
          onValueChange={(value) =>
            onDraftCountryChange(value === ALL_OPTION_VALUE ? '' : value)
          }
        />

        <label className="admin-label">
          Budget min ($)
          <input
            className="admin-input"
            disabled={fetching}
            inputMode="decimal"
            placeholder="0.00"
            title="Server filter; applied when field loses focus"
            value={draftBudgetMinUsd}
            onBlur={onBudgetFiltersApply}
            onChange={(event) => onDraftBudgetMinUsdChange(event.target.value)}
          />
        </label>
        <label className="admin-label">
          Budget max ($)
          <input
            className="admin-input"
            disabled={fetching}
            inputMode="decimal"
            placeholder="0.00"
            title="Server filter; applied when field loses focus"
            value={draftBudgetMaxUsd}
            onBlur={onBudgetFiltersApply}
            onChange={(event) => onDraftBudgetMaxUsdChange(event.target.value)}
          />
        </label>

        <DateRangePicker
          className="admin-label--range"
          disabled={fetching}
          from={draftStatsFrom}
          id="campaign-list-stats-range"
          label="Period"
          to={draftStatsTo}
          variant="admin"
          onChange={onStatsRangeChange}
        />

        <CampaignListColumnsMenu
          columnPrefs={columnPrefs}
          disabled={fetching}
          onColumnPrefsChange={onColumnPrefsChange}
          rowDisplayPrefs={rowDisplayPrefs}
          onRowDisplayPrefsChange={onRowDisplayPrefsChange}
        />

        <Button
          type="button"
          variant="outline"
          disabled={fetching}
          title="Reset columns, widths, and row highlight preferences"
          onClick={onResetWorkspaceClick}
        >
          Reset view
        </Button>
      </div>
    </div>
  );
}
