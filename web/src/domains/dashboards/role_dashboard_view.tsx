import { RefreshCw } from 'lucide-react';
import { Link } from 'react-router-dom';

import { Button } from '@/components/ui/button';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import { PageLayout } from '@/shell/page_layout';
import type { CustomerComboboxOption } from '@/shell/customer_combobox';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { StubBanner } from '@/shell/stub_banner';
import { DASHBOARD_ROLES, formatDashboardRoleLabel } from '@/api/dashboards_api';
import type { DashboardRole } from '@/api/types';
import { CampaignsListFilterSelect } from '@/domains/campaigns/list/campaigns_list_filter_select';
import { BuyerDashboardToolbar, type BuyerDashboardCampaignOption } from '@/domains/dashboards/buyer_dashboard_toolbar';
import { BuyerDashboardView } from '@/domains/dashboards/buyer_dashboard_view';
import { parseBuyerPortfolio, type DashboardRangePreset } from '@/domains/dashboards/buyer_dashboard_types';
import { opsStatusTone } from '@/domains/ops/ops_status';
import { useBuyerDashboardPreferences } from '@/hooks/use_buyer_dashboard_preferences';
import { cn } from '@/lib/utils';

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
  draftFrom: string;
  draftTo: string;
  rangePreset: DashboardRangePreset;
  customerOptions: CustomerComboboxOption[];
  campaignOptions: BuyerDashboardCampaignOption[];
  payload: Record<string, unknown> | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  licenseGated: boolean;
  showApply: boolean;
  onDraftRoleChange: (role: DashboardRole) => void;
  onDraftCustomerIdChange: (value: string) => void;
  onDraftCampaignIdChange: (value: string) => void;
  onDraftRangeChange: (from: string, to: string) => void;
  onRangePresetChange: (preset: DashboardRangePreset) => void;
  onDraftFromChange: (value: string) => void;
  onDraftToChange: (value: string) => void;
  onApply: () => void;
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
    <span className={cn('text-xs text-zinc-500 dark:text-zinc-400', stale ? opsStatusTone('warn') : opsStatusTone('ok'))}>
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
  fetching,
  onDraftRoleChange,
  onDraftCustomerIdChange,
  onRangePresetChange,
  onDraftFromChange,
  onDraftToChange,
  onApply,
}: {
  role: DashboardRole;
  draftRole: DashboardRole;
  draftCustomerId: string;
  draftFrom: string;
  draftTo: string;
  rangePreset: DashboardRangePreset;
  customerOptions: CustomerComboboxOption[];
  fetching: boolean;
  onDraftRoleChange: (role: DashboardRole) => void;
  onDraftCustomerIdChange: (value: string) => void;
  onRangePresetChange: (preset: DashboardRangePreset) => void;
  onDraftFromChange: (value: string) => void;
  onDraftToChange: (value: string) => void;
  onApply: () => void;
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

  return (
    <div className="flex flex-col gap-3 flex flex-col gap-2">
      <div
        aria-label="Dashboard filters"
        className="flex flex-wrap items-center gap-2 flex flex-wrap items-center gap-2"
        role="search"
      >
        <CampaignsListFilterSelect
          aria-label="Role"
          options={roleOptions}
          value={draftRole}
          onValueChange={(value) => onDraftRoleChange(value as DashboardRole)}
        />
        <CampaignsListFilterSelect
          aria-label="Customer"
          disabled={fetching}
          options={customerSelectOptions}
          value={draftCustomerId || ALL_OPTION_VALUE}
          onValueChange={(value) =>
            onDraftCustomerIdChange(value === ALL_OPTION_VALUE ? '' : value)
          }
        />
        <CampaignsListFilterSelect
          aria-label="Range preset"
          options={RANGE_PRESET_OPTIONS}
          value={rangePreset}
          onValueChange={(value) => onRangePresetChange(value as DashboardRangePreset)}
        />
        <DatetimePicker
          disabled={fetching || rangePreset !== 'custom'}
          id="dashboard-from"
          label="From"
          value={draftFrom}
          onChange={onDraftFromChange}
        />
        <DatetimePicker
          disabled={fetching || rangePreset !== 'custom'}
          id="dashboard-to"
          label="To"
          value={draftTo}
          onChange={onDraftToChange}
        />
        <Button disabled={fetching || !draftCustomerId.trim()} type="button" onClick={onApply}>
          Load
        </Button>
      </div>
      {role !== 'buyer' ? (
        <p className="text-zinc-500 dark:text-zinc-400">
          <Link className="text-blue-600 hover:underline dark:text-blue-400" to="/rtb">
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
  draftFrom,
  draftTo,
  rangePreset,
  customerOptions,
  campaignOptions,
  payload,
  fetching,
  error,
  hasSnapshot,
  licenseGated,
  onDraftRoleChange,
  onDraftCustomerIdChange,
  onDraftCampaignIdChange,
  onDraftRangeChange,
  onDraftFromChange,
  onDraftToChange,
  onRangePresetChange,
  onApply,
  showApply,
  clickLogHref,
}: RoleDashboardViewProps) {
  const { preferences, applyPreferences } = useBuyerDashboardPreferences();

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (licenseGated) {
    return (
      <PageLayout title="Dashboards">
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

  const buyerPortfolio = role === 'buyer' ? parseBuyerPortfolio(payload) : undefined;
  const pageTitle = role === 'buyer' ? 'Dashboard' : `${formatDashboardRoleLabel(role)} dashboard`;

  return (
    <PageLayout
      badge={freshnessBadge(payload)}
      controlPanel={
        role === 'buyer' ? (
          <BuyerDashboardToolbar
              campaignOptions={campaignOptions}
              customerOptions={customerOptions}
              draftCampaignId={draftCampaignId}
              draftCustomerId={draftCustomerId}
              draftFrom={draftFrom}
              draftTo={draftTo}
              fetching={fetching}
              preferences={preferences}
              showApply={showApply}
              onApply={onApply}
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
              fetching={fetching}
              rangePreset={rangePreset}
              role={role}
              onApply={onApply}
              onDraftCustomerIdChange={onDraftCustomerIdChange}
              onDraftFromChange={onDraftFromChange}
              onDraftRoleChange={onDraftRoleChange}
              onDraftToChange={onDraftToChange}
              onRangePresetChange={onRangePresetChange}
            />
        )
      }
      headerActions={
        <Button
          aria-label="Refresh dashboard"
          disabled={fetching || !draftCustomerId.trim()}
          loading={fetching}
          size="icon"
          type="button"
          variant="secondary"
          onClick={onApply}
        >
          <RefreshCw aria-hidden className="h-4 w-4" />
        </Button>
      }
      title={pageTitle}
    >
      <div className="flex flex-col gap-3">
        {buyerPortfolio ? (
          <BuyerDashboardView
            clickLogHref={clickLogHref}
            portfolio={buyerPortfolio}
            preferences={preferences}
          />
        ) : null}

        {error && hasSnapshot ? <ErrorBlock error={error} title="Refresh failed" /> : null}
      </div>
    </PageLayout>
  );
}
