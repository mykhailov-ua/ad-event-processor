import { Link } from 'react-router-dom';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import { PageLayout } from '@/shell/page_layout';
import type { CustomerComboboxOption } from '@/shell/customer_combobox';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { StubBanner } from '@/shell/stub_banner';
import { DASHBOARD_ROLES, formatDashboardRoleLabel } from '@/api/dashboards_api';
import type { DashboardRole } from '@/api/types';
import {
  CampaignsListFilterSelect,
  CampaignsListSearchableFilterSelect,
} from '@/domains/campaigns/list/campaigns_list_filter_select';
import {
  BuyerDashboardToolbar,
  type BuyerDashboardCampaignOption,
} from '@/domains/dashboards/buyer_dashboard_toolbar';
import { BuyerDashboardView } from '@/domains/dashboards/buyer_dashboard_view';
import {
  dashboardFilterFieldClass,
  dashboardFilterLabelClass,
  dashboardPageWorkspaceClass,
} from '@/domains/dashboards/dashboard_classes';
import {
  parseBuyerPortfolio,
  type DashboardRangePreset,
} from '@/domains/dashboards/buyer_dashboard_types';
import { DASHBOARD_BUYER_PAGE_DESCRIPTION } from '@/domains/dashboards/dashboard_preferences';
import { opsStatusTone } from '@/domains/ops/ops_status';
import { useBuyerDashboardPreferences } from '@/hooks/use_buyer_dashboard_preferences';
import {
  DirectoryFilterForm,
  FilterField,
  FilterPanel,
  FILTER_PANEL_FLAT_CLASS,
} from '@/shell/filter_panel';
import { ListRefreshBand } from '@/shell/list_refresh_band';
import { cn } from '@/lib/utils';

// L3 dashboard shell: loading/error/stale-while-revalidate (EH-SI2).
// licenseGated -> StubBanner; blocking load -> PageSkeleton; refresh error with snapshot -> ErrorBlock band.
export type { DashboardRangePreset };

const ALL_OPTION_VALUE = '__all__';

const RANGE_PRESET_OPTIONS = [
  { value: 'today', label: 'Today' },
  { value: '7d', label: 'Last 7 days' },
  { value: '30d', label: 'Last 30 days' },
  { value: 'custom', label: 'Custom' },
];

export type RoleDashboardViewProps = {
  role: DashboardRole;
  draftRole: DashboardRole;
  draftCustomerId: string;
  draftCampaignId: string;
  appliedCustomerId: string;
  appliedCampaignId: string;
  appliedFrom: string;
  appliedTo: string;
  draftFrom: string;
  draftTo: string;
  rangePreset: DashboardRangePreset;
  customerOptions: CustomerComboboxOption[];
  campaignOptions: BuyerDashboardCampaignOption[];
  payload: Record<string, unknown> | undefined;
  fetching: boolean;
  dashboardRevalidating?: boolean;
  dashboardLastUpdatedAt?: string | null;
  error: Error | undefined;
  hasSnapshot: boolean;
  customerRequired: boolean;
  licenseGated: boolean;
  onDraftRoleChange: (role: DashboardRole) => void;
  onDraftCustomerIdChange: (value: string) => void;
  onDraftCampaignIdChange: (value: string) => void;
  onDraftRangeChange: (from: string, to: string) => void;
  onRangePresetChange: (preset: DashboardRangePreset) => void;
  onDraftFromChange: (value: string) => void;
  onDraftToChange: (value: string) => void;
  onRefresh: () => void;
  clickLogHref?: string;
};

function freshnessBadge(payload: Record<string, unknown> | undefined) {
  const portfolio = parseBuyerPortfolio(payload);
  const label = portfolio?.kpis?.freshness?.label ?? portfolio?.fraud?.freshness?.label;
  if (!label) {
    return undefined;
  }
  const stale = portfolio?.kpis?.freshness?.stale === true;
  return (
    <span
      className={cn(
        'text-xs text-muted-foreground',
        stale ? opsStatusTone('warn') : opsStatusTone('ok')
      )}
    >
      {label}
    </span>
  );
}

function RoleDashboardFilters({
  role,
  draftRole,
  draftCustomerId,
  draftFrom,
  draftTo,
  rangePreset,
  customerOptions,
  onDraftRoleChange,
  onDraftCustomerIdChange,
  onRangePresetChange,
  onDraftFromChange,
  onDraftToChange,
}: {
  role: DashboardRole;
  draftRole: DashboardRole;
  draftCustomerId: string;
  draftFrom: string;
  draftTo: string;
  rangePreset: DashboardRangePreset;
  customerOptions: CustomerComboboxOption[];
  onDraftRoleChange: (role: DashboardRole) => void;
  onDraftCustomerIdChange: (value: string) => void;
  onRangePresetChange: (preset: DashboardRangePreset) => void;
  onDraftFromChange: (value: string) => void;
  onDraftToChange: (value: string) => void;
}) {
  const roleOptions = DASHBOARD_ROLES.map((item) => ({
    value: item,
    label: formatDashboardRoleLabel(item),
  }));

  const customerSelectOptions = [
    { value: ALL_OPTION_VALUE, label: 'All customers' },
    ...customerOptions.map((customer) => ({
      value: customer.id,
      label: customer.name,
    })),
  ];

  const customRange = rangePreset === 'custom';

  return (
    <div className="grid gap-2">
      <FilterPanel aria-label="Dashboard filters" className={FILTER_PANEL_FLAT_CLASS} role="search">
        <DirectoryFilterForm layout="campaigns">
          <FilterField
            className={dashboardFilterFieldClass}
            label="Role"
            labelClassName={dashboardFilterLabelClass}
          >
            <CampaignsListFilterSelect
              aria-label="Role"
              options={roleOptions}
              value={draftRole}
              onValueChange={(value) => onDraftRoleChange(value as DashboardRole)}
            />
          </FilterField>

          <FilterField
            className={dashboardFilterFieldClass}
            label="Customer"
            labelClassName={dashboardFilterLabelClass}
          >
            <CampaignsListSearchableFilterSelect
              aria-label="Customer"
              options={customerSelectOptions}
              searchPlaceholder="All customers"
              value={draftCustomerId || ALL_OPTION_VALUE}
              onValueChange={(value) =>
                onDraftCustomerIdChange(value === ALL_OPTION_VALUE ? '' : value)
              }
            />
          </FilterField>

          <FilterField
            className={dashboardFilterFieldClass}
            label="Range"
            labelClassName={dashboardFilterLabelClass}
          >
            <CampaignsListFilterSelect
              aria-label="Range preset"
              options={RANGE_PRESET_OPTIONS}
              value={rangePreset}
              onValueChange={(value) => onRangePresetChange(value as DashboardRangePreset)}
            />
          </FilterField>

          <FilterField
            className={dashboardFilterFieldClass}
            htmlFor="dashboard-from"
            label="From"
            labelClassName={dashboardFilterLabelClass}
          >
            <DatetimePicker
              className="[&>label]:sr-only"
              disabled={!customRange}
              id="dashboard-from"
              label="From"
              value={draftFrom}
              onChange={(value) => {
                if (!customRange) {
                  toast.message('Switch range preset to Custom to edit dates');
                  return;
                }
                onDraftFromChange(value);
              }}
            />
          </FilterField>

          <FilterField
            className={dashboardFilterFieldClass}
            htmlFor="dashboard-to"
            label="To"
            labelClassName={dashboardFilterLabelClass}
          >
            <DatetimePicker
              className="[&>label]:sr-only"
              disabled={!customRange}
              id="dashboard-to"
              label="To"
              value={draftTo}
              onChange={(value) => {
                if (!customRange) {
                  toast.message('Switch range preset to Custom to edit dates');
                  return;
                }
                onDraftToChange(value);
              }}
            />
          </FilterField>
        </DirectoryFilterForm>
      </FilterPanel>

      {role !== 'buyer' ? (
        <p className="text-sm text-muted-foreground">
          <Link className="text-primary hover:underline" to="/rtb">
            RTB overview
          </Link>
        </p>
      ) : null}
    </div>
  );
}

export function RoleDashboardView({
  role,
  draftRole,
  draftCustomerId,
  draftCampaignId,
  appliedCustomerId,
  appliedCampaignId,
  appliedFrom,
  appliedTo,
  draftFrom,
  draftTo,
  rangePreset,
  customerOptions,
  campaignOptions,
  payload,
  fetching,
  dashboardRevalidating = false,
  dashboardLastUpdatedAt = null,
  error,
  hasSnapshot,
  customerRequired,
  licenseGated,
  onDraftRoleChange,
  onDraftCustomerIdChange,
  onDraftCampaignIdChange,
  onDraftRangeChange,
  onDraftFromChange,
  onDraftToChange,
  onRangePresetChange,
  onRefresh,
  clickLogHref,
}: RoleDashboardViewProps) {
  const { preferences, applyPreferences } = useBuyerDashboardPreferences();

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (licenseGated) {
    return (
      <PageLayout fillViewport title="Dashboards" workspaceClassName={dashboardPageWorkspaceClass}>
        <StubBanner
          message="This dashboard is not available on the current license tier."
          title="License required"
        />
      </PageLayout>
    );
  }

  if (error && !hasSnapshot) {
    return (
      <ErrorBlock
        error={error}
        title={`Could not load ${formatDashboardRoleLabel(role)} dashboard`}
      />
    );
  }

  const buyerPortfolio =
    role === 'buyer' && !customerRequired ? parseBuyerPortfolio(payload) : undefined;
  const pageTitle = role === 'buyer' ? 'Dashboard' : `${formatDashboardRoleLabel(role)} dashboard`;
  const pageDescription = role === 'buyer' ? DASHBOARD_BUYER_PAGE_DESCRIPTION : undefined;

  return (
    <PageLayout
      badge={freshnessBadge(payload)}
      description={pageDescription}
      fillViewport
      controlPanel={
        role === 'buyer' ? (
          <BuyerDashboardToolbar
            campaignOptions={campaignOptions}
            customerOptions={customerOptions}
            draftCampaignId={draftCampaignId}
            draftCustomerId={draftCustomerId}
            draftFrom={draftFrom}
            draftTo={draftTo}
            preferences={preferences}
            onDraftCampaignIdChange={onDraftCampaignIdChange}
            onDraftCustomerIdChange={onDraftCustomerIdChange}
            onDraftRangeChange={onDraftRangeChange}
            onPreferencesApply={applyPreferences}
          />
        ) : (
          <RoleDashboardFilters
            customerOptions={customerOptions}
            draftCustomerId={draftCustomerId}
            draftFrom={draftFrom}
            draftRole={draftRole}
            draftTo={draftTo}
            rangePreset={rangePreset}
            role={role}
            onDraftCustomerIdChange={onDraftCustomerIdChange}
            onDraftFromChange={onDraftFromChange}
            onDraftRoleChange={onDraftRoleChange}
            onDraftToChange={onDraftToChange}
            onRangePresetChange={onRangePresetChange}
          />
        )
      }
      headerActions={
        <ListRefreshBand
          ariaLabel="Refresh dashboard"
          disabled={customerRequired}
          lastUpdatedAt={dashboardLastUpdatedAt}
          loading={fetching || dashboardRevalidating}
          onRefresh={onRefresh}
          title={customerRequired ? 'Select a customer' : 'Refresh dashboard'}
        />
      }
      title={pageTitle}
      workspaceClassName={dashboardPageWorkspaceClass}
    >
      <div className="grid min-w-0 gap-3">
        {customerRequired ? (
          <EmptyState
            description="Choose a customer group to load KPIs, charts, and breakdown tables."
            title="Select a customer"
            variant="blank-slate"
          />
        ) : null}

        {buyerPortfolio ? (
          <BuyerDashboardView
            clickLogHref={clickLogHref}
            customerId={appliedCustomerId}
            periodFrom={appliedFrom}
            periodTo={appliedTo}
            portfolio={buyerPortfolio}
            preferences={preferences}
            scopedCampaignId={appliedCampaignId || undefined}
          />
        ) : null}

        {error && hasSnapshot ? <ErrorBlock error={error} title="Refresh failed" /> : null}
      </div>
    </PageLayout>
  );
}
