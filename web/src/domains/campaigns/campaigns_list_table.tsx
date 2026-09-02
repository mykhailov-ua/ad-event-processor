import { useCallback, useState, type DragEvent } from 'react';
import { GripVertical } from 'lucide-react';
import { Link } from 'react-router-dom';

import { Checkbox } from '@/components/ui/checkbox';
import {
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu';
import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin } from '@/api/types';
import {
  campaignListConversionsLabel,
  campaignListCostLabel,
  campaignListClicksLabel,
  campaignListMarginCostLabel,
  campaignListProfitLabel,
  campaignListRevenueLabel,
  formatCampaignListCr,
  formatCampaignListRoi,
  sumCampaignListTotals,
  type CampaignListTotals,
} from '@/domains/campaigns/campaign_list_format';
import {
  CAMPAIGN_LIST_COLUMN_LABELS,
  isCampaignListColumnDraggable,
  isCampaignListColumnResizable,
  middleColumnsForSettings,
  moveDataColumn,
  type CampaignListColumnId,
  type CampaignListColumnPrefs,
  type CampaignListMiddleColumnId,
  type CampaignListReorderableColumnId,
  saveCampaignListColumnPrefs,
  setMiddleColumnVisible,
  visibleCampaignListColumns,
} from '@/domains/campaigns/campaign_list_columns';
import type { CampaignSortField, SortOrder } from '@/domains/campaigns/campaigns_list_types';
import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/campaign_metrics_shared';
import { formatDashboardUsdFromMicro } from '@/domains/dashboards/dashboard_format';
import { useCampaignListColumnResize } from '@/domains/campaigns/use_campaign_list_column_resize';
import { cn } from '@/lib/utils';

const COLUMN_DRAG_MIME = 'application/x-aed-campaign-column';

function campaignDisplayId(id: string): string {
  let hash = 0;
  for (let index = 0; index < id.length; index += 1) {
    hash = (hash * 31 + id.charCodeAt(index)) >>> 0;
  }
  return String((hash % 9000) + 1000);
}

function statusDotClass(status: string): string {
  const normalized = status.toUpperCase();
  if (normalized === 'PAUSED') {
    return 'is-paused';
  }
  if (normalized === 'ARCHIVED') {
    return 'is-archived';
  }
  return 'is-active';
}

function isNumericColumn(id: CampaignListColumnId): boolean {
  return id !== 'select' && id !== 'name' && id !== 'source' && id !== 'group';
}

function sortFieldForColumn(id: CampaignListColumnId): CampaignSortField | undefined {
  switch (id) {
    case 'name':
      return 'name';
    case 'cost':
      return 'spend';
    default:
      return undefined;
  }
}

export type CampaignsListTableProps = {
  items: Campaign[];
  customerNameById: Record<string, string>;
  metricsById: Record<string, CampaignListMetrics>;
  marginsById: Record<string, CampaignMargin>;
  columnPrefs: CampaignListColumnPrefs;
  columnWidths: Record<CampaignListColumnId, number>;
  onColumnPrefsChange: (prefs: CampaignListColumnPrefs) => void;
  selectedIds: Set<string>;
  onSelectedIdsChange: (ids: Set<string>) => void;
  appliedSort: CampaignSortField;
  appliedOrder: SortOrder;
  onColumnSort: (field: CampaignSortField) => void;
  onColumnWidthCommit: (columnId: CampaignListColumnId, widthPx: number) => void;
  fetching?: boolean;
  emptyMessage?: string;
  onCreateClick?: () => void;
};

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

  if (columnId === 'select') {
    return <span className="sr-only">Select</span>;
  }

  const labelNode =
    sortField != null ? (
      <button
        className={cn(
          'min-w-0 flex-1 truncate text-left',
          active && 'text-[var(--campaigns-ws-text)]',
        )}
        disabled={disabled}
        type="button"
        onClick={() => onColumnSort(sortField)}
      >
        <span className="truncate">{label}</span>
        <span aria-hidden className="ml-1 text-[0.625rem] leading-none">
          {active ? (appliedOrder === 'asc' ? '▲' : '▼') : '↕'}
        </span>
      </button>
    ) : (
      <span className="min-w-0 flex-1 truncate">{label}</span>
    );

  return (
    <div
      className={cn(
        'campaigns-list-workspace-th-inner',
        dragOver && 'campaigns-list-workspace-th-inner--drag-over',
      )}
      onDragEnd={onDragEnd}
      onDragEnter={onDragEnter}
      onDragLeave={onDragLeave}
      onDragOver={onDragOver}
      onDrop={onDrop}
    >
      {draggable ? (
        <button
          aria-label={`Reorder ${label} column`}
          className="campaigns-list-col-grip"
          draggable
          type="button"
          onDragStart={(event) => {
            event.dataTransfer.setData(COLUMN_DRAG_MIME, columnId);
            event.dataTransfer.effectAllowed = 'move';
            onDragStart();
          }}
        >
          <GripVertical aria-hidden className="h-3 w-3 shrink-0 opacity-40" />
        </button>
      ) : null}
      {labelNode}
    </div>
  );
}

function MiddleCell({
  columnId,
  campaign,
  row,
  metrics,
  margin,
  customerName,
}: {
  columnId: CampaignListMiddleColumnId;
  campaign: Campaign;
  row: CampaignWithMoneyDisplay;
  metrics?: CampaignListMetrics;
  margin?: CampaignMargin;
  customerName: string;
}) {
  switch (columnId) {
    case 'source':
      return campaign.traffic_template_id ? (
        <span className="campaigns-list-workspace-link">{campaign.traffic_template_id}</span>
      ) : (
        <span className="text-[var(--campaigns-ws-muted)]">Direct</span>
      );
    case 'flows':
      return campaign.flow_id ? (
        <Link className="campaigns-list-workspace-link" to={`/flows/${campaign.flow_id}`}>
          1
        </Link>
      ) : (
        '0'
      );
    case 'clicks':
      return campaignListClicksLabel(metrics) || '0';
    case 'conversions':
      return campaignListConversionsLabel(metrics) || '0';
    case 'cr':
      return formatCampaignListCr(metrics?.clicks, metrics?.conversions) || '0.00%';
    case 'revenue':
      return campaignListRevenueLabel(margin) || campaignListCostLabel(row) || '—';
    case 'cost':
      return campaignListMarginCostLabel(margin) || campaignListCostLabel(row) || '—';
    case 'profit':
      return campaignListProfitLabel(margin) || '—';
    case 'roi': {
      const roi = formatCampaignListRoi(margin?.operator_margin_micro, margin?.rtb_cost_micro);
      return roi ? <span className="campaigns-list-workspace-roi">{roi}</span> : '—';
    }
    case 'group':
      return (
        <Link className="campaigns-list-workspace-link" to={`/customers/${campaign.customer_id}`}>
          {customerName}
        </Link>
      );
    default:
      return null;
  }
}

function TotalsCell({
  columnId,
  totals,
  pageCount,
}: {
  columnId: CampaignListColumnId;
  totals: CampaignListTotals;
  pageCount: number;
}) {
  switch (columnId) {
    case 'select':
    case 'id':
      return null;
    case 'name':
      return <span className="font-semibold">Total</span>;
    case 'flows':
      return String(totals.flows);
    case 'clicks':
      return String(totals.clicks);
    case 'conversions':
      return String(totals.conversions);
    case 'cr':
      return formatCampaignListCr(totals.clicks, totals.conversions) || '0.00%';
    case 'revenue':
      return formatDashboardUsdFromMicro(totals.revenueMicro) || '—';
    case 'cost':
      return formatDashboardUsdFromMicro(totals.costMicro) || '—';
    case 'profit':
      return formatDashboardUsdFromMicro(totals.profitMicro) || '—';
    case 'roi': {
      const roi = formatCampaignListRoi(totals.profitMicro, totals.costMicro);
      return roi ? <span className="campaigns-list-workspace-roi">{roi}</span> : '—';
    }
    case 'group':
      return `${pageCount} on page`;
    default:
      return null;
  }
}

export function CampaignsListTable({
  items,
  customerNameById,
  metricsById,
  marginsById,
  columnPrefs,
  columnWidths,
  onColumnPrefsChange,
  selectedIds,
  onSelectedIdsChange,
  appliedSort,
  appliedOrder,
  onColumnSort,
  onColumnWidthCommit,
  fetching = false,
  emptyMessage = 'No campaigns match the current filters.',
  onCreateClick,
}: CampaignsListTableProps) {
  const columns = visibleCampaignListColumns(columnPrefs);
  const { localWidths, startResize } = useCampaignListColumnResize({
    columnWidths,
    onColumnWidthCommit,
  });
  const tableWidthPx = columns.reduce((sum, columnId) => sum + localWidths[columnId], 0);
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

  function persistColumnPrefs(next: CampaignListColumnPrefs) {
    onColumnPrefsChange(next);
    saveCampaignListColumnPrefs(next);
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
      <div className="flex flex-1 flex-col items-center justify-center gap-3 bg-white px-6 py-16 text-center">
        <p className="max-w-md text-sm text-[var(--campaigns-ws-muted)]">{emptyMessage}</p>
        {onCreateClick ? (
          <button className="campaigns-list-workspace-btn-create" type="button" onClick={onCreateClick}>
            Create
          </button>
        ) : null}
      </div>
    );
  }

  return (
    <div className="campaigns-list-workspace-table-wrap ui-scrollbar">
      <div className="campaigns-list-workspace-table-frame">
        <div aria-hidden className="campaigns-list-workspace-table-accent" />
        <table
          className="campaigns-list-workspace-table"
          style={{ width: `${tableWidthPx}px` }}
        >
        <colgroup>
          {columns.map((columnId) => (
            <col key={columnId} style={{ width: `${localWidths[columnId]}px` }} />
          ))}
        </colgroup>
        <thead>
          <tr>
            {columns.map((columnId) => {
              const draggable = isCampaignListColumnDraggable(columnId);
              const resizable = isCampaignListColumnResizable(columnId);
              const reorderableTarget = draggable ? columnId : null;

              return (
                <th
                  key={columnId}
                  className={cn(
                    'campaigns-list-workspace-th',
                    isNumericColumn(columnId) && 'campaigns-list-workspace-num',
                    draggingColumnId === columnId && 'campaigns-list-workspace-th--dragging',
                  )}
                >
                  {columnId === 'select' ? (
                    <div className="campaigns-list-workspace-th-inner">
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
                      dragOver={dragOverColumnId === reorderableTarget}
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
                      className="campaigns-list-col-resize-handle"
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
        </thead>
        <tbody>
          {items.map((campaign) => {
            const row = campaign as CampaignWithMoneyDisplay;
            const metrics = metricsById[campaign.id];
            const margin = marginsById[campaign.id];
            const customerName = customerNameById[campaign.customer_id] ?? campaign.customer_id;

            return (
              <tr key={campaign.id}>
                {columns.map((columnId) => {
                  if (columnId === 'select') {
                    return (
                      <td key={columnId}>
                        <Checkbox
                          aria-label={`Select ${campaign.name}`}
                          checked={selectedIds.has(campaign.id)}
                          disabled={fetching}
                          onCheckedChange={(checked) =>
                            toggleOne(campaign.id, checked === true)
                          }
                        />
                      </td>
                    );
                  }

                  if (columnId === 'id') {
                    return (
                      <td key={columnId} className="campaigns-list-workspace-num" title={campaign.id}>
                        {campaignDisplayId(campaign.id)}
                      </td>
                    );
                  }

                  if (columnId === 'name') {
                    return (
                      <td key={columnId}>
                        <div className="flex min-w-0 items-center gap-1">
                          <span
                            aria-hidden
                            className={cn(
                              'campaigns-list-workspace-status-dot shrink-0',
                              statusDotClass(campaign.status),
                            )}
                          />
                          <Link
                            className="campaigns-list-workspace-link min-w-0 truncate font-medium"
                            title={campaign.name}
                            to={`/campaigns/${campaign.id}/edit`}
                          >
                            {campaign.name}
                          </Link>
                        </div>
                      </td>
                    );
                  }

                  return (
                    <td
                      key={columnId}
                      className={cn(isNumericColumn(columnId) && 'campaigns-list-workspace-num')}
                    >
                      <MiddleCell
                        campaign={campaign}
                        columnId={columnId}
                        customerName={customerName}
                        margin={margin}
                        metrics={metrics}
                        row={row}
                      />
                    </td>
                  );
                })}
              </tr>
            );
          })}
        </tbody>
        <tfoot>
          <tr>
            {columns.map((columnId) => (
              <td
                key={columnId}
                className={cn(isNumericColumn(columnId) && 'campaigns-list-workspace-num')}
              >
                <TotalsCell columnId={columnId} pageCount={items.length} totals={totals} />
              </td>
            ))}
          </tr>
        </tfoot>
      </table>
      </div>
    </div>
  );
}

export type CampaignsListColumnSettingsProps = {
  columnPrefs: CampaignListColumnPrefs;
  onColumnPrefsChange: (prefs: CampaignListColumnPrefs) => void;
  onImportClick?: () => void;
  onWizardClick?: () => void;
};

export function CampaignsListColumnSettings({
  columnPrefs,
  onColumnPrefsChange,
  onImportClick,
  onWizardClick,
}: CampaignsListColumnSettingsProps) {
  function persist(next: CampaignListColumnPrefs) {
    onColumnPrefsChange(next);
    saveCampaignListColumnPrefs(next);
  }

  return (
    <DropdownMenuContent align="end" className="w-72" plain>
      <div className="border-b border-[#dddddd] px-3 py-2 text-sm font-medium text-[#333333]">Columns</div>
      <div className="tracker-plain-menu-scrollbar max-h-72 overflow-y-auto p-2">
        {middleColumnsForSettings(columnPrefs).map((columnId) => (
          <label
            key={columnId}
            className="flex cursor-pointer items-center gap-2 rounded-none px-1 py-1 text-sm text-[#333333] hover:bg-[#f0f0f0]"
          >
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
      {onWizardClick || onImportClick ? (
        <>
          <DropdownMenuSeparator className="bg-[#dddddd]" />
          {onImportClick ? (
            <DropdownMenuItem plain onClick={onImportClick}>
              Import
            </DropdownMenuItem>
          ) : null}
          {onWizardClick ? (
            <DropdownMenuItem plain onClick={onWizardClick}>
              Wizard
            </DropdownMenuItem>
          ) : null}
          <DropdownMenuItem asChild plain>
            <Link to="/docs">Documentation</Link>
          </DropdownMenuItem>
        </>
      ) : null}
    </DropdownMenuContent>
  );
}
