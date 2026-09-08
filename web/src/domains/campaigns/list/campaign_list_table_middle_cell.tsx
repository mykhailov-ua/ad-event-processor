import { Link } from 'react-router-dom';

import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignStatsQuery } from '@/api/types';
import { CampaignCountryBadges } from '@/domains/campaigns/list/campaign_country_badges';
import { CampaignMetricsPopover } from '@/domains/campaigns/list/campaign_metrics_popover';
import type { CampaignListMiddleColumnId } from '@/domains/campaigns/list/campaign_list_columns';
import { rateBenchmarkToneClass } from '@/domains/campaigns/list/campaign_list_rate_tone';
import type { CampaignRowVm } from '@/domains/campaigns/list/campaign_list_row_vm';
import {
  RateMetricCell,
  tableCellClass,
} from '@/domains/campaigns/list/campaign_list_table_cell_format';
import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/list/campaign_metrics_shared';
import { cn } from '@/lib/utils';

export type CampaignListTableMiddleCellProps = {
  columnId: CampaignListMiddleColumnId;
  campaign: Campaign;
  vm: CampaignRowVm;
  marginBreach?: boolean;
  listMetrics?: CampaignListMetrics;
  onOpenOverview?: (campaign: Campaign) => void;
  statsCacheRevision: string;
  statsQuery?: CampaignStatsQuery;
};

export function CampaignListTableMiddleCell({
  columnId,
  campaign,
  vm,
  marginBreach = false,
  listMetrics,
  onOpenOverview,
  statsCacheRevision,
  statsQuery,
}: CampaignListTableMiddleCellProps) {
  const row = campaign as CampaignWithMoneyDisplay;

  switch (columnId) {
    case 'clicks':
      return <span className={tableCellClass(vm.clicks.isZero)}>{vm.clicks.text}</span>;
    case 'impressions':
      return <span className={tableCellClass(vm.impressions.isZero)}>{vm.impressions.text}</span>;
    case 'ctr':
      if (!vm.ctr) {
        return <span className={tableCellClass(true)}>-</span>;
      }
      return (
        <RateMetricCell isEmpty={false} ratePct={vm.ctr.valPct}>
          {vm.ctr.text}
        </RateMetricCell>
      );
    case 'unique_clicks':
      return <span className={tableCellClass(vm.uniqueClicks.isZero)}>{vm.uniqueClicks.text}</span>;
    case 'lp_clicks':
      return <span className={tableCellClass(vm.lpClicks.isZero)}>{vm.lpClicks.text}</span>;
    case 'lp_views':
      return <span className={tableCellClass(vm.lpViews.isZero)}>{vm.lpViews.text}</span>;
    case 'group':
      return (
        <Link
          className="block max-w-full whitespace-nowrap"
          title={vm.groupLabel ?? vm.groupCustomerId}
          to={`/customers/${vm.groupCustomerId}`}
          onClick={(event) => event.stopPropagation()}
        >
          {vm.groupLabel ?? vm.groupCustomerId}
        </Link>
      );
    case 'lp_ctr':
      if (!vm.lpCtr) {
        return <span className={tableCellClass(true)}>-</span>;
      }
      return (
        <RateMetricCell isEmpty={false} ratePct={vm.lpCtr.valPct}>
          {vm.lpCtr.text}
        </RateMetricCell>
      );
    case 'cr':
      return (
        <span
          className={tableCellClass(
            vm.cr.isZero,
            rateBenchmarkToneClass(vm.cr.isZero ? null : vm.cr.valPct)
          )}
        >
          {vm.cr.text}
        </span>
      );
    case 'leads':
      return (
        <span className={tableCellClass(vm.leads.isZero, undefined, 'conversion')}>
          {vm.leads.text}
        </span>
      );
    case 'approved':
      return (
        <span className={tableCellClass(vm.approved.isZero, undefined, 'approved')}>
          {vm.approved.text}
        </span>
      );
    case 'hold_leads':
      return <span className={tableCellClass(vm.holdLeads.isZero)}>{vm.holdLeads.text}</span>;
    case 'rejected_leads':
      return (
        <span className={tableCellClass(vm.rejectedLeads.isZero)}>{vm.rejectedLeads.text}</span>
      );
    case 'approve_rate':
      if (!vm.approveRate) {
        return <span className={tableCellClass(true)}>-</span>;
      }
      return (
        <RateMetricCell isEmpty={false} ratePct={vm.approveRate.valPct}>
          {vm.approveRate.text}
        </RateMetricCell>
      );
    case 'epc':
      return <span className={tableCellClass(vm.epc.isZero)}>{vm.epc.text}</span>;
    case 'cpc':
      return <span className={tableCellClass(vm.cpc.isZero)}>{vm.cpc.text}</span>;
    case 'cpa':
      return <span className={tableCellClass(vm.cpa.isZero)}>{vm.cpa.text}</span>;
    case 'ecpa':
      return <span className={tableCellClass(vm.ecpa.isZero)}>{vm.ecpa.text}</span>;
    case 'cpm':
      if (!vm.cpm) {
        return <span className={tableCellClass(true)}>-</span>;
      }
      return <span className={tableCellClass()}>{vm.cpm}</span>;
    case 'blocks':
      return <span className={tableCellClass(vm.blocks.isZero)}>{vm.blocks.text}</span>;
    case 'block_pct':
      if (!vm.blockPct) {
        return <span className={tableCellClass(true)}>-</span>;
      }
      return <span className={tableCellClass()}>{vm.blockPct}</span>;
    case 'bots':
      return <span className={tableCellClass(vm.bots.isZero)}>{vm.bots.text}</span>;
    case 'bot_pct':
      if (!vm.botPct) {
        return <span className={tableCellClass(true)}>-</span>;
      }
      return <span className={tableCellClass()}>{vm.botPct}</span>;
    case 'revenue':
      return (
        <span className={tableCellClass(vm.revenue.isZero, undefined, 'primary')}>
          {vm.revenue.text}
        </span>
      );
    case 'cost':
      return <span className={tableCellClass(vm.cost.isZero)}>{vm.cost.text}</span>;
    case 'profit':
      return (
        <span
          className={tableCellClass(
            vm.profit.isZero,
            vm.profitToneClass,
            vm.profit.isZero ? undefined : 'primary'
          )}
        >
          {vm.profit.text}
        </span>
      );
    case 'roi':
      return <span className={tableCellClass(vm.roi.isZero, vm.roiToneClass)}>{vm.roi.text}</span>;
    case 'budget_pct':
      if (vm.budgetPct == null) {
        return <span className={tableCellClass(true)}>-</span>;
      }
      return (
        <CampaignMetricsPopover
          campaign={row}
          listMetrics={listMetrics}
          onOpenOverview={onOpenOverview}
          statsCacheRevision={statsCacheRevision}
          statsQuery={statsQuery}
          triggerContent={<span className={tableCellClass()}>{vm.budgetPct.toFixed(1)}%</span>}
        />
      );
    case 'flow':
      return (
        <span className="text-muted-foreground" title={vm.flowId ?? ''}>
          {vm.flowId ? vm.flowId.slice(0, 8) : '-'}
        </span>
      );
    case 'owner':
      return (
        <span
          className="block max-w-full whitespace-nowrap text-muted-foreground"
          title={vm.ownerId}
        >
          {vm.ownerLabel}
        </span>
      );
    case 'countries':
      if (vm.countries.length === 0) {
        return <span className={tableCellClass(true)}>-</span>;
      }
      return (
        <span className="block max-w-full">
          <CampaignCountryBadges
            className="max-w-full"
            compact
            countries={vm.countries}
            max={3}
            overflowMenu
          />
        </span>
      );
    default:
      return null;
  }
}
