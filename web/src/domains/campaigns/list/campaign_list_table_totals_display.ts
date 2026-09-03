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
import type { CampaignListColumnId } from '@/domains/campaigns/list/campaign_list_columns';

// Width probe strings must match CampaignListTableTotalsCell output.
export function campaignListTotalsCellDisplayText(
  columnId: CampaignListColumnId,
  totals: CampaignListTotals,
  funnelTotals: CampaignFunnelCounts,
  pageCount: number,
  totalsLabel = 'Total',
): string {
  switch (columnId) {
    case 'select':
    case 'id':
      return '';
    case 'name':
      return totalsLabel;
    case 'clicks':
      return formatTableCount(totals.clicks).text;
    case 'ctr':
      if (totals.impressions <= 0 || totals.clicks <= 0) {
        return '-';
      }
      return formatSourceCtr(totals.clicks, totals.impressions);
    case 'lp_ctr':
      if (totals.clicks <= 0 || funnelTotals.lpClicks <= 0) {
        return '-';
      }
      return formatLpCtr(funnelTotals.lpClicks, totals.clicks);
    case 'leads':
      return formatTableCount(funnelTotals.rawLeads).text;
    case 'approved':
      return formatTableCount(funnelTotals.approved).text;
    case 'hold_leads':
      return formatTableCount(funnelTotals.hold).text;
    case 'rejected_leads':
      return formatTableCount(funnelTotals.rejected).text;
    case 'approve_rate':
      if (funnelTotals.rawLeads <= 0) {
        return '-';
      }
      return formatApproveRate(funnelTotals.approved, funnelTotals.rawLeads);
    case 'lp_clicks':
      return formatTableCount(funnelTotals.lpClicks).text;
    case 'lp_views':
      return formatTableCount(funnelTotals.lpViews).text;
    case 'bots':
      return formatTableCount(funnelTotals.bots).text;
    case 'impressions':
    case 'unique_clicks':
    case 'blocks':
      return '';
    case 'cr':
      return formatTableCr(totals.clicks, funnelTotals.approved).text;
    case 'cpm':
      if (totals.impressions <= 0 || totals.costMicro <= 0) {
        return '-';
      }
      return formatCpmUsd(totals.costMicro, totals.impressions);
    case 'block_pct':
      if (totals.clicks <= 0 || totals.blocks <= 0) {
        return '-';
      }
      return formatRelativeRate(totals.blocks, totals.clicks);
    case 'bot_pct':
      if (totals.clicks <= 0 || funnelTotals.bots <= 0) {
        return '-';
      }
      return formatRelativeRate(funnelTotals.bots, totals.clicks);
    case 'ecpa': {
      const ecpaMicro =
        funnelTotals.approved > 0 ? Math.trunc(totals.costMicro / funnelTotals.approved) : 0;
      return formatTableMoneyFromMicro(ecpaMicro).text;
    }
    case 'epc': {
      const epcMicro = totals.clicks > 0 ? Math.trunc(totals.revenueMicro / totals.clicks) : 0;
      return formatTableMoneyFromMicro(epcMicro).text;
    }
    case 'cpc': {
      const cpcMicro = totals.clicks > 0 ? Math.trunc(totals.costMicro / totals.clicks) : 0;
      return formatTableMoneyFromMicro(cpcMicro).text;
    }
    case 'cpa': {
      const cpaMicro =
        funnelTotals.rawLeads > 0 ? Math.trunc(totals.costMicro / funnelTotals.rawLeads) : 0;
      return formatTableMoneyFromMicro(cpaMicro).text;
    }
    case 'revenue': {
      const res = formatTableMoneyFromMicro(totals.revenueMicro);
      return res.isZero ? '0.00' : res.text;
    }
    case 'cost': {
      const res = formatTableMoneyFromMicro(totals.costMicro);
      return res.isZero ? '0.00' : res.text;
    }
    case 'profit': {
      const profitRes = formatTableMoneyFromMicro(totals.profitMicro);
      return profitRes.isZero ? '0.00' : profitRes.text;
    }
    case 'roi': {
      const roiRes = formatTableRoi(totals.profitMicro, totals.costMicro);
      return roiRes.isZero ? '0%' : roiRes.text;
    }
    case 'group':
      return `${pageCount} on page`;
    default:
      return '';
  }
}
