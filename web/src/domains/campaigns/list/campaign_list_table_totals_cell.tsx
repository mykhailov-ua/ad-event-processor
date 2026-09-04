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
import { profitToneClassFromMicro, roiToneClassFromRate } from '@/domains/campaigns/list/campaign_list_tone';
import type { CampaignListColumnId } from '@/domains/campaigns/list/campaign_list_columns';
import { RateMetricCell, tableCellClass } from '@/domains/campaigns/list/campaign_list_table_cell_format';

export type CampaignListTableTotalsCellProps = {
  columnId: CampaignListColumnId;
  totals: CampaignListTotals;
  funnelTotals: CampaignFunnelCounts;
  pageCount: number;
  totalsLabel?: string;
};

export function CampaignListTableTotalsCell({
  columnId,
  totals,
  funnelTotals,
  pageCount,
  totalsLabel = 'Total',
}: CampaignListTableTotalsCellProps) {
  switch (columnId) {
    case 'select':
    case 'id':
      return null;
    case 'name':
      return <span className="tabular-nums">{totalsLabel}</span>;
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
      const ecpaMicro =
        funnelTotals.approved > 0 ? Math.trunc(totals.costMicro / funnelTotals.approved) : 0;
      const res = formatTableMoneyFromMicro(ecpaMicro);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'epc': {
      const epcMicro = totals.clicks > 0 ? Math.trunc(totals.revenueMicro / totals.clicks) : 0;
      const res = formatTableMoneyFromMicro(epcMicro);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'cpc': {
      const cpcMicro = totals.clicks > 0 ? Math.trunc(totals.costMicro / totals.clicks) : 0;
      const res = formatTableMoneyFromMicro(cpcMicro);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'cpa': {
      const cpaMicro =
        funnelTotals.rawLeads > 0 ? Math.trunc(totals.costMicro / funnelTotals.rawLeads) : 0;
      const res = formatTableMoneyFromMicro(cpaMicro);
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
      const profitTone = profitToneClassFromMicro(totals.profitMicro);
      return (
        <span
          className={tableCellClass(
            profitRes.isZero,
            profitRes.isZero ? undefined : profitTone,
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
        <span className={tableCellClass(roiRes.isZero, roiToneClassFromRate(roiRes))}>
          {roiRes.isZero ? '0%' : roiRes.text}
        </span>
      );
    }
    case 'group':
      return <span className="tabular-nums text-muted-foreground">{pageCount} on page</span>;
    default:
      return null;
  }
}
