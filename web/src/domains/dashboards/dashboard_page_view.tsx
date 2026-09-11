import { Link } from 'react-router-dom';

import type { AdopsDashboardPayload, BuyerDashboardPayload } from '@/domains/dashboards/dashboard_types';
import { DashboardAttentionList } from '@/domains/dashboards/dashboard_attention_list';
import { DashboardBreakdownList } from '@/domains/dashboards/dashboard_breakdown_list';
import { DashboardKpiBlocks } from '@/domains/dashboards/dashboard_kpi_blocks';
import { DashboardNav } from '@/domains/dashboards/dashboard_nav';
import { DashboardStaleBanner } from '@/domains/dashboards/dashboard_stale_banner';
import { DashboardAdopsCampaignsList } from '@/domains/dashboards/dashboard_adops_campaigns_list';
import { DashboardWorstSourcesList } from '@/domains/dashboards/dashboard_worst_sources_list';
import type { DashboardPageWorkspace } from '@/domains/dashboards/use_dashboard_page_workspace';
import { DASHBOARD_RANGE_PRESETS } from '@/domains/dashboards/use_dashboard_page_workspace';
import { Button } from '@/components/ui/button';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { adminSpacing, adminTypography, opsControlPanelClass } from '@/lib/admin_spacing';
import { ErrorBlock } from '@/shell/error_block';
import { PageChrome } from '@/shell/page_chrome';
import { PageSectionStack } from '@/shell/page_layout';
import { PageSkeleton } from '@/shell/page_skeleton';
import { BentoSection } from '@/shell/bento_card';
import {
  DirectoryFilterForm,
  FilterField,
  FilterPanel,
  INLINE_FILTER_ACTION_GRID_TWO_ACTIONS_CLASS,
} from '@/shell/filter_panel';

type DashboardPageViewProps = DashboardPageWorkspace & {
  title: string;
  description: string;
};

export function DashboardPageView({
  title,
  description,
  role,
  draftCustomerId,
  draftFrom,
  draftTo,
  draftPreset,
  data,
  fetching,
  revalidating,
  error,
  hasSnapshot,
  sessionStaleBanner,
  exportTrueRoiHref,
  exportCampaignOverviewHref,
  onDraftCustomerIdChange,
  onDraftFromChange,
  onDraftToChange,
  onDraftPresetChange,
  onApply,
  onRefresh,
}: DashboardPageViewProps) {
  const kpis = data?.kpis;
  const stale = kpis?.freshness?.stale === true;
  const buyerData = role === 'buyer' ? (data as BuyerDashboardPayload | undefined) : undefined;
  const adopsData = role === 'adops' ? (data as AdopsDashboardPayload | undefined) : undefined;

  if (fetching && !hasSnapshot) {
    return <PageSkeleton />;
  }

  return (
    <PageChrome
      description={description}
      title={title}
      actions={
        <div className={adminSpacing.flex.buttonGroup}>
          <Button asChild data-testid="dashboard-export-true-roi" type="button" variant="secondary">
            <Link to={exportTrueRoiHref}>Export true-roi</Link>
          </Button>
          <Button asChild data-testid="dashboard-open-campaigns" type="button" variant="secondary">
            <Link to="/campaigns">Open campaigns</Link>
          </Button>
        </div>
      }
      controlPanel={
        <div className={opsControlPanelClass}>
          <DashboardNav />
          <FilterPanel aria-label="Dashboard filters" data-testid="dashboard-filters">
            <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
              <FilterField htmlFor="dashboard-customer-id" label="Customer ID">
                <Input
                  data-testid="dashboard-customer-id"
                  id="dashboard-customer-id"
                  value={draftCustomerId}
                  onChange={(event) => onDraftCustomerIdChange(event.target.value)}
                />
              </FilterField>
              <FilterField htmlFor="dashboard-preset" label="Preset">
                <Select
                  value={draftPreset}
                  onValueChange={(value) => onDraftPresetChange(value as typeof draftPreset)}
                >
                  <SelectTrigger data-testid="dashboard-preset" id="dashboard-preset">
                    <SelectValue placeholder="Range preset" />
                  </SelectTrigger>
                  <SelectContent>
                    {DASHBOARD_RANGE_PRESETS.map((preset) => (
                      <SelectItem key={preset.id} value={preset.id}>
                        {preset.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FilterField>
              <FilterField htmlFor="dashboard-from" label="From">
                <DatetimePicker
                  data-testid="dashboard-from"
                  id="dashboard-from"
                  value={draftFrom}
                  onChange={onDraftFromChange}
                />
              </FilterField>
              <FilterField htmlFor="dashboard-to" label="To">
                <DatetimePicker
                  data-testid="dashboard-to"
                  id="dashboard-to"
                  value={draftTo}
                  onChange={onDraftToChange}
                />
              </FilterField>
              <div className={INLINE_FILTER_ACTION_GRID_TWO_ACTIONS_CLASS}>
                <Button data-testid="dashboard-apply" type="button" onClick={onApply}>
                  Apply dates
                </Button>
                <Button
                  data-testid="dashboard-refresh"
                  disabled={revalidating}
                  type="button"
                  variant="secondary"
                  onClick={onRefresh}
                >
                  Refresh
                </Button>
              </div>
            </DirectoryFilterForm>
          </FilterPanel>
        </div>
      }
    >
      {error ? <ErrorBlock error={error} title="Could not load dashboard" /> : null}

      {!error && hasSnapshot ? (
        <PageSectionStack>
          <DashboardStaleBanner freshness={kpis?.freshness} sessionStaleBanner={sessionStaleBanner} />
          {kpis?.freshness?.freshness_label ? (
            <p className={adminTypography.bodyMuted} data-testid="dashboard-freshness-label">
              {kpis.freshness.freshness_label}
            </p>
          ) : null}
          <BentoSection data-testid="dashboard-kpi-section" title="KPIs">
            <DashboardKpiBlocks
              kpis={kpis}
              portfolioCounts={
                buyerData
                  ? {
                      active: buyerData.active,
                      paused: buyerData.paused,
                      archived: buyerData.archived,
                      impressions7d: buyerData.impressions_7d,
                      clicks7d: buyerData.clicks_7d,
                      overspendCount: buyerData.overspend_count,
                    }
                  : undefined
              }
              stale={stale}
            />
          </BentoSection>
          {role === 'buyer' && buyerData ? (
            <>
              <DashboardAttentionList rows={buyerData.attention} />
              <DashboardBreakdownList
                exportHref={exportCampaignOverviewHref}
                stale={stale}
                table={buyerData.breakdowns?.campaigns}
              />
            </>
          ) : null}
          {role === 'adops' && adopsData ? (
            <>
              <DashboardAdopsCampaignsList
                exportHref={exportCampaignOverviewHref}
                rows={adopsData.campaigns}
                stale={stale}
                tableMeta={adopsData.table_sections_meta?.campaigns}
              />
              <DashboardWorstSourcesList rows={adopsData.worst_sources} />
            </>
          ) : null}
        </PageSectionStack>
      ) : null}
    </PageChrome>
  );
}
