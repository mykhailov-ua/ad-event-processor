import { ArrowDown, ArrowUp } from 'lucide-react';
import { useCallback, useState, type DragEvent } from 'react';
import { Link } from 'react-router-dom';

import { Checkbox } from '@/components/ui/checkbox';
import { Table, TableBody, TableFooter, TableHeader } from '@/components/ui/table';
import {
  DropdownMenuContent,
} from '@/components/ui/dropdown-menu';
import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin } from '@/api/types';
import {
  campaignListRevenueMicro,
  formatTableCount,
  formatTableCr,
  formatTableMoneyFromMicro,
  formatTableRoi,
  parseAndFormatTableMoneyStr,
  sumCampaignListTotals,
  type CampaignListTotals,
} from '@/domains/campaigns/list/campaign_list_format';
import {
  formatApproveRate,
  formatCpmUsd,
  formatLpCtr,
  formatRelativeRate,
  formatSourceCtr,
  resolveCampaignFunnelCounts,
  sumCampaignFunnelTotals,
  type CampaignFunnelCounts,
} from '@/domains/campaigns/list/campaign_list_funnel';
import {
  CAMPAIGN_LIST_COLUMN_LABELS,
  CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX,
  clampCampaignListColumnWidthPx,
  isCampaignListColumnDraggable,
  isCampaignListColumnResizable,
  middleColumnsForSettings,
  moveDataColumn,
  COLUMN_DRAG_MIME,
  type CampaignListColumnId,
  type CampaignListColumnPrefs,
  type CampaignListMiddleColumnId,
  type CampaignListReorderableColumnId,
  saveCampaignListColumnPrefs,
  setMiddleColumnVisible,
  visibleCampaignListColumns,
} from '@/domains/campaigns/list/campaign_list_columns';
import { CampaignCountryBadges } from '@/domains/campaigns/list/campaign_country_badges';
import { percentRate, rateBenchmarkToneClass } from '@/domains/campaigns/list/campaign_list_rate_tone';
import {
  campaignListRowClass,
  campaignListRowStatusEdgeClass,
  campaignStatusBadgeClass,
} from '@/domains/campaigns/list/campaign_list_row_tone';
import type { CampaignSortField, SortOrder } from '@/domains/campaigns/list/campaigns_list_types';
import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/list/campaign_metrics_shared';
import { useCampaignListColumnResize } from '@/domains/campaigns/list/use_campaign_list_column_resize';
import { formatCampaignStatusLabel } from '@/lib/admin_typography';
import { cn } from '@/lib/utils';

import { campaignDisplayId } from '@/domains/campaigns/list/campaign_display_id';

function isNumericColumn(id: CampaignListColumnId): boolean {
  return (
    id === 'clicks' ||
    id === 'impressions' ||
    id === 'ctr' ||
    id === 'unique_clicks' ||
    id === 'lp_clicks' ||
    id === 'lp_views' ||
    id === 'lp_ctr' ||
    id === 'cr' ||
    id === 'leads' ||
    id === 'approved' ||
    id === 'hold_leads' ||
    id === 'rejected_leads' ||
    id === 'approve_rate' ||
    id === 'epc' ||
    id === 'cpc' ||
    id === 'cpa' ||
    id === 'ecpa' ||
    id === 'cpm' ||
    id === 'blocks' ||
    id === 'block_pct' ||
    id === 'bots' ||
    id === 'bot_pct' ||
    id === 'revenue' ||
    id === 'cost' ||
    id === 'profit' ||
    id === 'roi' ||
    id === 'budget_pct'
  );
}

function sortFieldForColumn(id: CampaignListColumnId): CampaignSortField | undefined {
  switch (id) {
    case 'name':
      return 'name';
    case 'clicks':
      return 'clicks';
    case 'cost':
      return 'spend';
    default:
      if (isNumericColumn(id)) {
        return id as CampaignSortField;
      }
      return undefined;
  }
}

function rowToneClass(
  campaign: CampaignWithMoneyDisplay,
  margin: CampaignMargin | undefined,
  selected: boolean,
  highlightActiveRows: boolean,
): string {
  return cn(
    campaignListRowClass({
      status: campaign.status,
      statusTone: campaign.status_tone,
      selected,
      highlightActiveRows,
      margin,
    }),
    campaignListRowStatusEdgeClass(campaign.status, campaign.status_tone),
  );
}

function tableCellClass(
  isZero?: boolean,
  extra?: string,
  emphasis?: 'primary' | 'secondary' | 'conversion',
): string {
  return cn(
    'admin-table-cell num',
    isZero && 'admin-metric-zero',
    emphasis === 'primary' && !isZero && 'admin-table-metric-primary',
    emphasis === 'secondary' && !isZero && 'admin-table-metric-secondary',
    emphasis === 'conversion' && !isZero && 'admin-table-metric-conversion',
    extra,
  );
}

function RateMetricCell({
  children,
  isEmpty,
  ratePct,
}: {
  children: string;
  isEmpty: boolean;
  ratePct: number | null;
}) {
  return (
    <span className={tableCellClass(isEmpty, rateBenchmarkToneClass(isEmpty ? null : ratePct))}>
      {children}
    </span>
  );
}

function profitToneClass(margin?: CampaignMargin): string {
  const profitMicro = margin?.operator_margin_micro;
  if (profitMicro == null || profitMicro === 0) {
    return 'admin-metric-zero';
  }
  return profitMicro > 0
    ? 'admin-positive'
    : 'admin-negative';
}

function roiToneClass(margin?: CampaignMargin): string {
  const roi = formatTableRoi(margin?.operator_margin_micro, margin?.rtb_cost_micro);
  if (roi.isZero) {
    return 'admin-metric-zero';
  }
  return roi.valPct >= 0
    ? 'admin-positive'
    : 'admin-negative';
}

export type CampaignsListTableProps = {
  items: Campaign[];
  customerNameById: Record<string, string>;
  ownerEmailById: Record<string, string>;
  metricsById: Record<string, CampaignListMetrics>;
  marginsById: Record<string, CampaignMargin>;
  columnPrefs: CampaignListColumnPrefs;
  columnWidths: Record<CampaignListColumnId, number>;
  highlightActiveRows?: boolean;
  onColumnPrefsChange: (prefs: CampaignListColumnPrefs) => void;
  selectedIds: Set<string>;
  onSelectedIdsChange: (ids: Set<string>) => void;
  appliedSort: CampaignSortField;
  appliedOrder: SortOrder;
  onColumnSort: (field: CampaignSortField) => void;
  onColumnWidthCommit: (columnId: CampaignListColumnId, widthPx: number) => void;
  fetching?: boolean;
  emptyMessage?: string;
  onCampaignOverview?: (campaign: Campaign) => void;
};

function resolveColumnWidthPx(
  columnId: CampaignListColumnId,
  localWidths: Readonly<Partial<Record<CampaignListColumnId, number>>>,
): number {
  const width = localWidths[columnId];
  if (width != null && Number.isFinite(width) && width > 0) {
    return clampCampaignListColumnWidthPx(columnId, width);
  }
  return CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[columnId];
}

function HeaderCell({
  columnId,
  appliedSort,
  appliedOrder,
  onColumnSort,
  disabled,
  draggable,
  dragOver,
  onDragStart,
  onDragEnter,
  onDragOver,
  onDragLeave,
  onDrop,
  onDragEnd,
}: {
  columnId: CampaignListColumnId;
  appliedSort: CampaignSortField;
  appliedOrder: SortOrder;
  onColumnSort: (field: CampaignSortField) => void;
  disabled?: boolean;
  draggable: boolean;
  dragOver: boolean;
  onDragStart: () => void;
  onDragEnter: () => void;
  onDragOver: (event: DragEvent<HTMLTableCellElement>) => void;
  onDragLeave: () => void;
  onDrop: (event: DragEvent<HTMLTableCellElement>) => void;
  onDragEnd: () => void;
}) {
  const sortField = sortFieldForColumn(columnId);
  const active = sortField != null && appliedSort === sortField;
  const label = CAMPAIGN_LIST_COLUMN_LABELS[columnId];
  const isNum = isNumericColumn(columnId);

  if (columnId === 'select') {
    return <span className="sr-only">Select</span>;
  }

  return (
    <>
      <div
        className={cn('admin-table-th-inner', isNum && 'num', dragOver && 'bg-[var(--admin-item-highlight)]')}
        onDragEnd={onDragEnd}
        onDragEnter={onDragEnter}
        onDragLeave={onDragLeave}
        onDragOver={onDragOver}
        onDrop={onDrop}
      >
        {sortField != null ? (
          <button
            className={cn('admin-table-th-sort', isNum && 'num', active && 'is-active')}
            disabled={disabled}
            title={label}
            type="button"
            onClick={() => onColumnSort(sortField)}
          >
            {label}
            {active ? (
              appliedOrder === 'asc' ? (
                <ArrowUp aria-hidden className="ml-0.5 h-3 w-3 shrink-0" />
              ) : (
                <ArrowDown aria-hidden className="ml-0.5 h-3 w-3 shrink-0" />
              )
            ) : null}
          </button>
        ) : (
          <span className={cn('admin-table-th-label', isNum && 'num')} title={label}>
            {label}
          </span>
        )}
      </div>
      {draggable ? (
        <button
          aria-label={`Reorder ${label} column`}
          className="admin-col-grip"
          draggable
          type="button"
          onDragStart={(event) => {
            event.dataTransfer.setData(COLUMN_DRAG_MIME, columnId);
            event.dataTransfer.effectAllowed = 'move';
            onDragStart();
          }}
        >
          ::
        </button>
      ) : null}
    </>
  );
}

function MiddleCell({
  columnId,
  campaign,
  row,
  metrics,
  margin,
  customerName,
  ownerEmailById,
}: {
  columnId: CampaignListMiddleColumnId;
  campaign: Campaign;
  row: CampaignWithMoneyDisplay;
  metrics?: CampaignListMetrics;
  margin?: CampaignMargin;
  customerName: string;
  ownerEmailById: Record<string, string>;
}) {
  const clicks = metrics?.clicks ?? 0;
  const impressions = metrics?.impressions ?? 0;
  const costMicro = margin?.rtb_cost_micro ?? 0;
  const revenueMicro = campaignListRevenueMicro(margin);
  const funnel = resolveCampaignFunnelCounts(metrics);
  const blocks = metrics?.blocks ?? 0;

  switch (columnId) {
    case 'status': {
      const label = formatCampaignStatusLabel(campaign.status, row.status_label);
      return (
        <span className={campaignStatusBadgeClass(campaign.status, row.status_tone)} title={label}>
          {label}
        </span>
      );
    }
    case 'tags':
      return <span className="admin-table-cell admin-table-metric-secondary">-</span>;
    case 'clicks': {
      const res = formatTableCount(clicks);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'impressions': {
      const res = formatTableCount(impressions);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'ctr': {
      if (impressions <= 0 || clicks <= 0) {
        return <span className={tableCellClass(true)}>-</span>;
      }
      return (
        <RateMetricCell isEmpty={false} ratePct={percentRate(clicks, impressions)}>
          {formatSourceCtr(clicks, impressions)}
        </RateMetricCell>
      );
    }
    case 'unique_clicks': {
      const res = formatTableCount(metrics?.unique_clicks);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'lp_clicks': {
      const res = formatTableCount(funnel.lpClicks);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'lp_views': {
      const res = formatTableCount(funnel.lpViews);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'group':
      return (
        <Link
          className="admin-table-cell"
          title={customerName}
          to={`/customers/${campaign.customer_id}`}
          onClick={(event) => event.stopPropagation()}
        >
          {customerName}
        </Link>
      );
    case 'lp_ctr': {
      if (clicks <= 0 || funnel.lpClicks <= 0) {
        return <span className={tableCellClass(true)}>-</span>;
      }
      return (
        <RateMetricCell isEmpty={false} ratePct={percentRate(funnel.lpClicks, clicks)}>
          {formatLpCtr(funnel.lpClicks, clicks)}
        </RateMetricCell>
      );
    }
    case 'cr': {
      const res = formatTableCr(clicks, funnel.approved);
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
    case 'leads': {
      const res = formatTableCount(funnel.rawLeads);
      return <span className={tableCellClass(res.isZero, undefined, 'conversion')}>{res.text}</span>;
    }
    case 'approved': {
      const res = formatTableCount(funnel.approved);
      return <span className={tableCellClass(res.isZero, undefined, 'primary')}>{res.text}</span>;
    }
    case 'hold_leads': {
      const res = formatTableCount(funnel.hold);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'rejected_leads': {
      const res = formatTableCount(funnel.rejected);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'approve_rate': {
      if (funnel.rawLeads <= 0) {
        return <span className={tableCellClass(true)}>-</span>;
      }
      return (
        <RateMetricCell
          isEmpty={false}
          ratePct={percentRate(funnel.approved, funnel.rawLeads)}
        >
          {formatApproveRate(funnel.approved, funnel.rawLeads)}
        </RateMetricCell>
      );
    }
    case 'epc': {
      const epcMicro = clicks > 0 ? Math.trunc(revenueMicro / clicks) : 0;
      const res = formatTableMoneyFromMicro(epcMicro);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'cpc': {
      const cpcMicro = clicks > 0 ? Math.trunc(costMicro / clicks) : 0;
      const res = formatTableMoneyFromMicro(cpcMicro);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'cpa': {
      const cpaMicro = funnel.rawLeads > 0 ? Math.trunc(costMicro / funnel.rawLeads) : 0;
      const res = formatTableMoneyFromMicro(cpaMicro);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'ecpa': {
      const ecpaMicro = funnel.approved > 0 ? Math.trunc(costMicro / funnel.approved) : 0;
      const res = formatTableMoneyFromMicro(ecpaMicro);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'cpm': {
      if (impressions <= 0 || costMicro <= 0) {
        return <span className={tableCellClass(true)}>-</span>;
      }
      return <span className={tableCellClass()}>{formatCpmUsd(costMicro, impressions)}</span>;
    }
    case 'blocks': {
      const res = formatTableCount(blocks);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'block_pct': {
      if (clicks <= 0 || blocks <= 0) {
        return <span className={tableCellClass(true)}>-</span>;
      }
      return <span className={tableCellClass()}>{formatRelativeRate(blocks, clicks)}</span>;
    }
    case 'bots': {
      const res = formatTableCount(funnel.bots);
      return <span className={tableCellClass(res.isZero)}>{res.text}</span>;
    }
    case 'bot_pct': {
      if (clicks <= 0 || funnel.bots <= 0) {
        return <span className={tableCellClass(true)}>-</span>;
      }
      return <span className={tableCellClass()}>{formatRelativeRate(funnel.bots, clicks)}</span>;
    }
    case 'revenue': {
      const res = revenueMicro > 0
        ? formatTableMoneyFromMicro(revenueMicro)
        : parseAndFormatTableMoneyStr(row.current_spend_display ?? row.current_spend);
      return <span className={tableCellClass(res.isZero, undefined, 'primary')}>{res.text}</span>;
    }
    case 'cost': {
      const costRes = costMicro > 0
        ? formatTableMoneyFromMicro(costMicro)
        : parseAndFormatTableMoneyStr(row.current_spend_display ?? row.current_spend);
      return <span className={tableCellClass(costRes.isZero)}>{costRes.text}</span>;
    }
    case 'profit': {
      const profitRes = formatTableMoneyFromMicro(margin?.operator_margin_micro);
      return (
        <span
          className={tableCellClass(
            profitRes.isZero,
            profitToneClass(margin),
            profitRes.isZero ? undefined : 'primary',
          )}
        >
          {profitRes.isZero ? '0.00' : profitRes.text}
        </span>
      );
    }
    case 'roi': {
      const roiRes = formatTableRoi(margin?.operator_margin_micro, margin?.rtb_cost_micro);
      return (
        <span className={tableCellClass(roiRes.isZero, roiToneClass(margin))}>
          {roiRes.isZero ? '0%' : roiRes.text}
        </span>
      );
    }
    case 'budget_pct': {
      const spendRes = parseAndFormatTableMoneyStr(row.current_spend_display ?? row.current_spend);
      const limitRes = parseAndFormatTableMoneyStr(row.budget_limit_display ?? row.budget_limit);
      if (limitRes.valUsd <= 0) {
        return <span className={tableCellClass(true)}>-</span>;
      }
      const pct = (spendRes.valUsd / limitRes.valUsd) * 100;
      return <span className={tableCellClass()}>{pct.toFixed(1)}%</span>;
    }
    case 'flow':
      return (
        <span className="admin-table-cell admin-table-metric-secondary" title={campaign.flow_id ?? ''}>
          {campaign.flow_id ? campaign.flow_id.slice(0, 8) : '-'}
        </span>
      );
    case 'owner': {
      const ownerId = campaign.owner_user_id ?? '';
      const label = ownerId ? (ownerEmailById[ownerId] ?? ownerId.slice(0, 8)) : '-';
      return (
        <span className="admin-table-cell admin-table-metric-secondary" title={ownerId}>
          {label}
        </span>
      );
    }
    case 'countries': {
      const countries = campaign.target_countries ?? [];
      if (countries.length === 0) {
        return <span className={tableCellClass(true)}>-</span>;
      }
      return (
        <span className="admin-table-cell">
          <CampaignCountryBadges countries={countries} max={5} />
        </span>
      );
    }
    default:
      return null;
  }
}

function TotalsCell({
  columnId,
  totals,
  funnelTotals,
  pageCount,
}: {
  columnId: CampaignListColumnId;
  totals: CampaignListTotals;
  funnelTotals: CampaignFunnelCounts;
  pageCount: number;
}) {
  switch (columnId) {
    case 'select':
    case 'id':
      return null;
    case 'name':
      return <span className="admin-table-cell">Total</span>;
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
                ? 'admin-positive'
                : 'admin-negative',
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
            roiRes.isZero ? undefined : roiRes.valPct >= 0 ? 'admin-positive' : 'admin-negative',
          )}
        >
          {roiRes.isZero ? '0%' : roiRes.text}
        </span>
      );
    }
    case 'group':
      return <span className="admin-table-cell admin-muted">{pageCount} on page</span>;
    default:
      return null;
  }
}

export function CampaignsListTable({
  items,
  customerNameById,
  ownerEmailById,
  metricsById,
  marginsById,
  columnPrefs,
  columnWidths,
  highlightActiveRows = false,
  onColumnPrefsChange,
  selectedIds,
  onSelectedIdsChange,
  appliedSort,
  appliedOrder,
  onColumnSort,
  onColumnWidthCommit,
  fetching = false,
  emptyMessage = 'No campaigns match the current filters.',
  onCampaignOverview,
}: CampaignsListTableProps) {
  const columns = visibleCampaignListColumns(columnPrefs);
  const { localWidths, startResize } = useCampaignListColumnResize({
    columnWidths,
    onColumnWidthCommit,
  });
  const columnWidthPxList = columns.map((columnId) => resolveColumnWidthPx(columnId, localWidths));
  const tableWidthPx = columnWidthPxList.reduce((sum, widthPx) => sum + widthPx, 0);
  const [draggingColumnId, setDraggingColumnId] = useState<CampaignListReorderableColumnId | null>(
    null,
  );
  const [dragOverColumnId, setDragOverColumnId] = useState<CampaignListReorderableColumnId | null>(
    null,
  );
  const allSelected = items.length > 0 && items.every((item) => selectedIds.has(item.id));
  const totals = sumCampaignListTotals(
    items as CampaignWithMoneyDisplay[],
    metricsById,
    marginsById,
  );
  const funnelTotals = sumCampaignFunnelTotals(items, metricsById);

  function toggleAll(checked: boolean) {
    if (!checked) {
      onSelectedIdsChange(new Set());
      return;
    }
    onSelectedIdsChange(new Set(items.map((item) => item.id)));
  }

  function toggleOne(campaignId: string, checked: boolean) {
    const next = new Set(selectedIds);
    if (checked) {
      next.add(campaignId);
    } else {
      next.delete(campaignId);
    }
    onSelectedIdsChange(next);
  }

  function isInteractiveRowTarget(target: EventTarget | null): boolean {
    if (!(target instanceof HTMLElement)) {
      return false;
    }
    return Boolean(
      target.closest('a, button, input, label, [role="checkbox"], .admin-col-resize, .admin-col-grip'),
    );
  }

  const handleColumnDrop = useCallback(
    (targetId: CampaignListReorderableColumnId, event: DragEvent<HTMLTableCellElement>) => {
      event.preventDefault();
      const raw = event.dataTransfer.getData(COLUMN_DRAG_MIME);
      if (!isCampaignListColumnDraggable(raw as CampaignListColumnId)) {
        return;
      }
      const draggedId = raw as CampaignListReorderableColumnId;
      if (draggedId === targetId) {
        return;
      }
      const next = {
        ...columnPrefs,
        dataColumnOrder: moveDataColumn(columnPrefs.dataColumnOrder, draggedId, targetId),
      };
      onColumnPrefsChange(next);
      saveCampaignListColumnPrefs(next);
      setDragOverColumnId(null);
      setDraggingColumnId(null);
    },
    [columnPrefs, onColumnPrefsChange],
  );

  const clearDragState = useCallback(() => {
    setDragOverColumnId(null);
    setDraggingColumnId(null);
  }, []);

  if (items.length === 0) {
    return (
      <div className="admin-panel">
        <p className="admin-muted">{emptyMessage}</p>
      </div>
    );
  }

  return (
    <div className="admin-table-wrap admin-campaigns-table-wrap">
      <Table
        bare
        className="admin-table admin-table--campaigns"
        style={{ width: `${tableWidthPx}px`, tableLayout: 'fixed' }}
      >
        <colgroup>
          {columns.map((columnId, index) => {
            const widthPx = columnWidthPxList[index] ?? CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[columnId];
            return <col key={columnId} style={{ width: `${widthPx}px` }} />;
          })}
        </colgroup>
        <TableHeader>
          <tr>
            {columns.map((columnId) => {
              const draggable = isCampaignListColumnDraggable(columnId);
              const resizable = isCampaignListColumnResizable(columnId);
              const reorderableTarget = draggable ? columnId : null;
              const isSelect = columnId === 'select';
              const isNum = isNumericColumn(columnId);

              return (
                <th
                  key={columnId}
                  className={cn(
                    isSelect ? 'admin-table-th--select' : isNum ? 'num' : undefined,
                    draggingColumnId === columnId && 'opacity-60',
                  )}
                >
                  {isSelect ? (
                    <div className="relative z-[2] flex h-full items-center justify-center">
                      <Checkbox
                        aria-label="Select all campaigns"
                        checked={allSelected}
                        disabled={fetching}
                        onCheckedChange={(checked) => toggleAll(checked === true)}
                      />
                    </div>
                  ) : (
                    <HeaderCell
                      appliedOrder={appliedOrder}
                      appliedSort={appliedSort}
                      columnId={columnId}
                      disabled={fetching}
                      dragOver={reorderableTarget != null && dragOverColumnId === reorderableTarget}
                      draggable={draggable}
                      onColumnSort={onColumnSort}
                      onDragEnd={clearDragState}
                      onDragEnter={() => {
                        if (reorderableTarget) {
                          setDragOverColumnId(reorderableTarget);
                        }
                      }}
                      onDragLeave={() => {
                        if (dragOverColumnId === reorderableTarget) {
                          setDragOverColumnId(null);
                        }
                      }}
                      onDragOver={(event) => {
                        if (!reorderableTarget) {
                          return;
                        }
                        event.preventDefault();
                        event.dataTransfer.dropEffect = 'move';
                        setDragOverColumnId(reorderableTarget);
                      }}
                      onDragStart={() => {
                        if (reorderableTarget) {
                          setDraggingColumnId(reorderableTarget);
                        }
                      }}
                      onDrop={(event) => {
                        if (reorderableTarget) {
                          handleColumnDrop(reorderableTarget, event);
                        }
                      }}
                    />
                  )}
                  {resizable ? (
                    <div
                      aria-label={`Resize ${CAMPAIGN_LIST_COLUMN_LABELS[columnId]} column`}
                      className="admin-col-resize"
                      role="separator"
                      onPointerDown={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        startResize(columnId, event.clientX);
                      }}
                    />
                  ) : null}
                </th>
              );
            })}
          </tr>
        </TableHeader>
        <TableBody>
          {items.map((campaign) => {
            const row = campaign as CampaignWithMoneyDisplay;
            const metrics = metricsById[campaign.id];
            const margin = marginsById[campaign.id];
            const customerName = customerNameById[campaign.customer_id] ?? campaign.customer_id;
            const selected = selectedIds.has(campaign.id);
            const toneClass = rowToneClass(row, margin, selected, highlightActiveRows);

            return (
              <tr
                key={campaign.id}
                className={cn(toneClass, 'cursor-pointer')}
                onClick={(event) => {
                  if (fetching || isInteractiveRowTarget(event.target)) {
                    return;
                  }
                  toggleOne(campaign.id, !selected);
                }}
              >
                {columns.map((columnId) => {
                  const isNum = isNumericColumn(columnId);

                  if (columnId === 'select') {
                    return (
                      <td key={columnId} className="admin-table-td--select">
                        <div className="relative z-[2] flex h-full items-center justify-center">
                          <Checkbox
                            aria-label={`Select ${campaign.name}`}
                            checked={selected}
                            disabled={fetching}
                            onCheckedChange={(checked) =>
                              toggleOne(campaign.id, checked === true)
                            }
                            onClick={(event) => event.stopPropagation()}
                          />
                        </div>
                      </td>
                    );
                  }

                  if (columnId === 'id') {
                    return (
                      <td key={columnId} className="num admin-table-td--id" title={campaign.id}>
                        <span className="admin-table-cell num admin-data-id admin-table-metric-secondary">
                          {campaignDisplayId(campaign)}
                        </span>
                      </td>
                    );
                  }

                  if (columnId === 'name') {
                    return (
                      <td key={columnId} className="admin-table-td--name">
                        <div className="admin-table-name">
                          <CampaignCountryBadges
                            className="admin-table-name__countries"
                            compact
                            countries={campaign.target_countries}
                            max={2}
                          />
                          <button
                            className="admin-table-name-link relative z-[2]"
                            title={campaign.name}
                            type="button"
                            onClick={(event) => {
                              event.stopPropagation();
                              onCampaignOverview?.(campaign);
                            }}
                          >
                            {campaign.name}
                          </button>
                          <Link
                            className="admin-table-row-action"
                            to={`/campaigns/${campaign.id}/edit`}
                            onClick={(event) => event.stopPropagation()}
                          >
                            Edit
                          </Link>
                        </div>
                      </td>
                    );
                  }

                  return (
                    <td key={columnId} className={isNum ? 'num' : undefined}>
                      <MiddleCell
                        campaign={campaign}
                        columnId={columnId as CampaignListMiddleColumnId}
                        customerName={customerName}
                        margin={margin}
                        metrics={metrics}
                        ownerEmailById={ownerEmailById}
                        row={row}
                      />
                    </td>
                  );
                })}
              </tr>
            );
          })}
        </TableBody>
        <TableFooter>
          <tr>
            {columns.map((columnId) => {
              const isNum = isNumericColumn(columnId);
              return (
                <td
                  key={columnId}
                  className={cn(
                    columnId === 'select' ? 'admin-table-td--select' : isNum ? 'num' : undefined,
                  )}
                >
                  <TotalsCell
                    columnId={columnId}
                    funnelTotals={funnelTotals}
                    pageCount={items.length}
                    totals={totals}
                  />
                </td>
              );
            })}
          </tr>
        </TableFooter>
      </Table>
    </div>
  );
}

export type CampaignsListColumnSettingsProps = {
  columnPrefs: CampaignListColumnPrefs;
  onColumnPrefsChange: (prefs: CampaignListColumnPrefs) => void;
};

export function CampaignsListColumnSettings({
  columnPrefs,
  onColumnPrefsChange,
}: CampaignsListColumnSettingsProps) {
  function persist(next: CampaignListColumnPrefs) {
    onColumnPrefsChange(next);
    saveCampaignListColumnPrefs(next);
  }

  return (
    <DropdownMenuContent align="end" className="w-72">
      <div className="admin-section-title">Columns</div>
      <div className="max-h-72 overflow-y-auto p-2">
        {middleColumnsForSettings(columnPrefs).map((columnId) => (
          <label key={columnId} className="flex cursor-pointer items-center gap-2 py-1">
            <Checkbox
              checked={!columnPrefs.hidden.includes(columnId)}
              onCheckedChange={(checked) =>
                persist({
                  ...columnPrefs,
                  hidden: setMiddleColumnVisible(columnPrefs.hidden, columnId, checked === true),
                })
              }
            />
            <span className="min-w-0 flex-1 truncate">{CAMPAIGN_LIST_COLUMN_LABELS[columnId]}</span>
          </label>
        ))}
      </div>
    </DropdownMenuContent>
  );
}
