import { Link } from 'react-router-dom';
import { RefreshCw } from 'lucide-react';

import { PageChrome } from '@/components/system/page_chrome';
import { PageToolbar } from '@/components/system/page_toolbar';
import { CustomerCombobox, type CustomerComboboxOption } from '@/components/system/customer_combobox';
import { FilterField } from '@/components/system/filter_panel';
import { ErrorBlock } from '@/components/system/error_block';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { StubBanner } from '@/components/system/stub_banner';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { DASHBOARD_ROLES, formatDashboardRoleLabel } from '@/api/dashboards_api';
import type { DashboardRole } from '@/api/types';
import { BuyerDashboardToolbar, type BuyerDashboardCampaignOption } from '@/domains/dashboards/buyer_dashboard_toolbar';
import { BuyerDashboardView } from '@/domains/dashboards/buyer_dashboard_view';
import { parseBuyerPortfolio, type DashboardRangePreset } from '@/domains/dashboards/buyer_dashboard_types';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import { useBuyerDashboardPreferences } from '@/hooks/use_buyer_dashboard_preferences';

export type { DashboardRangePreset };

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
  return <Badge variant={stale ? 'secondary' : 'outline'}>{label}</Badge>;
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
      <PageChrome title="Dashboards">
        <StubBanner
          title="License required"
          message="This dashboard is not available on the current license tier."
        />
      </PageChrome>
    );
  }

  if (error && !hasSnapshot) {
    return (
      <ErrorBlock
        title={`Could not load ${formatDashboardRoleLabel(role)} dashboard`}
        message={error.message}
      />
    );
  }

  const buyerPortfolio = role === 'buyer' ? parseBuyerPortfolio(payload) : undefined;

  return (
    <PageChrome
      title={role === 'buyer' ? 'Dashboard' : `${formatDashboardRoleLabel(role)} dashboard`}
      badge={freshnessBadge(payload)}
      actions={
        <Button
          disabled={fetching || !draftCustomerId.trim()}
          onClick={onApply}
          size="icon"
          type="button"
          variant="outline"
          aria-label="Refresh dashboard"
        >
          <RefreshCw className={fetching ? 'h-4 w-4 animate-spin' : 'h-4 w-4'} />
        </Button>
      }
    >
      {role === 'buyer' ? (
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
        <PageToolbar className="grid gap-4 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_minmax(0,1fr)_auto_auto] md:items-end">
          <div className="grid gap-2">
            <Label htmlFor="dashboard-role">Role</Label>
            <Select value={draftRole} onValueChange={(value) => onDraftRoleChange(value as DashboardRole)}>
              <SelectTrigger id="dashboard-role">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {DASHBOARD_ROLES.map((item) => (
                  <SelectItem key={item} value={item}>
                    {formatDashboardRoleLabel(item)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <FilterField htmlFor="dashboard-customer" label="Customer">
            <CustomerCombobox
              id="dashboard-customer"
              disabled={fetching}
              options={customerOptions}
              value={draftCustomerId}
              onValueChange={onDraftCustomerIdChange}
            />
          </FilterField>
          <div className="grid gap-2">
            <Label htmlFor="dashboard-range-preset">Range</Label>
            <Select
              value={rangePreset}
              onValueChange={(value) => onRangePresetChange(value as DashboardRangePreset)}
            >
              <SelectTrigger id="dashboard-range-preset">
                <SelectValue placeholder="Custom" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="today">Today</SelectItem>
                <SelectItem value="7d">Last 7 days</SelectItem>
                <SelectItem value="30d">Last 30 days</SelectItem>
                <SelectItem value="custom">Custom</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <DatetimePicker
            id="dashboard-from"
            label="From"
            value={draftFrom}
            onChange={onDraftFromChange}
          />
          <DatetimePicker id="dashboard-to" label="To" value={draftTo} onChange={onDraftToChange} />
          <Button
            className="md:col-span-5 md:justify-self-end"
            disabled={fetching || !draftCustomerId.trim()}
            onClick={onApply}
            type="button"
          >
            Load
          </Button>
        </PageToolbar>
      )}

      {role !== 'buyer' ? (
        <div className="text-sm text-muted-foreground">
          <Link className="hover:underline" to="/rtb">
            RTB overview
          </Link>
        </div>
      ) : null}

      {buyerPortfolio ? (
        <BuyerDashboardView
          clickLogHref={clickLogHref}
          portfolio={buyerPortfolio}
          preferences={preferences}
        />
      ) : null}

      {!buyerPortfolio && payload ? <JsonDashboardView payload={payload} /> : null}

      {error && hasSnapshot ? <ErrorBlock title="Refresh failed" message={error.message} /> : null}
    </PageChrome>
  );
}
