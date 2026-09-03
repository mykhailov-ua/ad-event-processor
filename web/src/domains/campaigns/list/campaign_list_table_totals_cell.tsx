import {
  formatApproveRate,
  formatCpmUsd,
  formatLpCtr,
  formatRelativeRate,
  formatSourceCtr,
  type CampaignFunnelCounts,
} from '@/domains/campaigns/list/campaign_list_funnel';
import {
  formatTableCount,
  formatTableCr,
  formatTableMoneyFromMicro,
  formatTableRoi,
  type CampaignListTotals,
} from '@/domains/campaigns/list/campaign_list_format';
import { percentRate, rateBenchmarkToneClass } from '@/domains/campaigns/list/campaign_list_rate_tone';
import type { CampaignListColumnId } from '@/domains/campaigns/list/campaign_list_columns';
import { RateMetricCell, tableCellClass } from '@/domains/campaigns/list/campaign_list_table_cell_format';

export type CampaignListTableTotalsCellProps = {
  columnId: CampaignListColumnId;
  totals: CampaignListTotals;
  funnelTotals: CampaignFunnelCounts;
  pageCount: number;
};

export function CampaignListTableTotalsCell({
  columnId,
  totals,
  funnelTotals,
  pageCount,
}: CampaignListTableTotalsCellProps) {
  switch (columnId) {
    case 'select':
    case 'id':
      return null;
    case 'name':
      return <span className="tabular-nums">Total</span>;
    case 'clicks': {
      const res = formatTableCount(totals.clicks);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'ctr': {
      if (totals.impressions <= 0 || totals.clicks <= 0) {
        return <span className={tableCellClass(true)}>-</span>;
      }
      return (
        <RateMetricCell isEmpty={false} ratePct={percentRate(totals.clicks, totals.impressions)}>
          {formatSourceCtr(totals.clicks, totals.impressions)}
        </RateMetricCell>
      );
    }
    case 'lp_ctr': {
      if (totals.clicks <= 0 || funnelTotals.lpClicks <= 0) {
        return <span className={tableCellClass(true)}>-</span>;
      }
      return (
        <RateMetricCell
          isEmpty={false}
          ratePct={percentRate(funnelTotals.lpClicks, totals.clicks)}
        >
          {formatLpCtr(funnelTotals.lpClicks, totals.clicks)}
        </RateMetricCell>
      );
    }
    case 'leads': {
      const res = formatTableCount(funnelTotals.rawLeads);
      return <span className={tableCellClass(res.isZero, undefined, 'conversion')}>{res.text}</span>;
    }
    case 'approved': {
      const res = formatTableCount(funnelTotals.approved);
      return <span className={tableCellClass(res.isZero, undefined, 'primary')}>{res.text}</span>;
    }
    case 'hold_leads': {
      const res = formatTableCount(funnelTotals.hold);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'rejected_leads': {
      const res = formatTableCount(funnelTotals.rejected);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'approve_rate': {
      if (funnelTotals.rawLeads <= 0) {
        return <span className={tableCellClass(true)}>-</span>;
      }
      return (
        <RateMetricCell
          isEmpty={false}
          ratePct={percentRate(funnelTotals.approved, funnelTotals.rawLeads)}
        >
          {formatApproveRate(funnelTotals.approved, funnelTotals.rawLeads)}
        </RateMetricCell>
      );
    }
    case 'lp_clicks': {
      const res = formatTableCount(funnelTotals.lpClicks);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'lp_views': {
      const res = formatTableCount(funnelTotals.lpViews);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'bots': {
      const res = formatTableCount(funnelTotals.bots);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'impressions':
    case 'unique_clicks':
    case 'blocks':
      return null;
    case 'cr': {
      const res = formatTableCr(totals.clicks, funnelTotals.approved);
      return (
        <span
          className={tableCellClass(
            res.isZero,
            rateBenchmarkToneClass(res.isZero ? null : res.valPct),
          )}
        >
          {res.text}
        </span>
      );
    }
    case 'cpm': {
      if (totals.impressions <= 0 || totals.costMicro <= 0) {
        return <span className={tableCellClass(true)}>-</span>;
      }
      return <span className={tableCellClass()}>{formatCpmUsd(totals.costMicro, totals.impressions)}</span>;
    }
    case 'block_pct': {
      if (totals.clicks <= 0 || totals.blocks <= 0) {
        return <span className={tableCellClass(true)}>-</span>;
      }
      return <span className={tableCellClass()}>{formatRelativeRate(totals.blocks, totals.clicks)}</span>;
    }
    case 'bot_pct': {
      if (totals.clicks <= 0 || funnelTotals.bots <= 0) {
        return <span className={tableCellClass(true)}>-</span>;
      }
      return <span className={tableCellClass()}>{formatRelativeRate(funnelTotals.bots, totals.clicks)}</span>;
    }
    case 'ecpa': {
      const ecpaMicro = funnelTotals.approved > 0 ? Math.trunc(totals.costMicro / funnelTotals.approved) : 0;
      const res = formatTableMoneyFromMicro(ecpaMicro);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'revenue': {
      const res = formatTableMoneyFromMicro(totals.revenueMicro);
      return (
        <span className={tableCellClass(res.isZero, undefined, 'primary')}>
          {res.isZero ? '0.00' : res.text}
        </span>
      );
    }
    case 'cost': {
      const res = formatTableMoneyFromMicro(totals.costMicro);
      return <span className={tableCellClass(res.isZero)}>{res.isZero ? '0.00' : res.text}</span>;
    }
    case 'profit': {
      const profitRes = formatTableMoneyFromMicro(totals.profitMicro);
      return (
        <span
          className={tableCellClass(
            profitRes.isZero,
            profitRes.isZero
              ? undefined
              : totals.profitMicro > 0
                ? 'font-semibold text-green-700 dark:text-green-400'
                : 'font-semibold text-red-700 dark:text-red-400',
            profitRes.isZero ? undefined : 'primary',
          )}
        >
          {profitRes.isZero ? '0.00' : profitRes.text}
        </span>
      );
    }
    case 'roi': {
      const roiRes = formatTableRoi(totals.profitMicro, totals.costMicro);
      return (
        <span
          className={tableCellClass(
            roiRes.isZero,
            roiRes.isZero
              ? undefined
              : roiRes.valPct >= 0
                ? 'font-semibold text-green-700 dark:text-green-400'
                : 'font-semibold text-red-700 dark:text-red-400',
          )}
        >
          {roiRes.isZero ? '0%' : roiRes.text}
        </span>
      );
    }
    case 'group':
      return <span className="tabular-nums text-zinc-500 dark:text-zinc-400">{pageCount} on page</span>;
    default:
      return null;
  }
}
