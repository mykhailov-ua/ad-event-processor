import { useCallback, useState, type DragEvent } from 'react';

import { Checkbox } from '@/components/ui/checkbox';
import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin, CampaignStatsQuery } from '@/api/types';
import {
  sumCampaignListTotals,
} from '@/domains/campaigns/list/campaign_list_format';
import { sumCampaignFunnelTotals } from '@/domains/campaigns/list/campaign_list_funnel';
import type { CampaignListFilterTotalsView } from '@/domains/campaigns/list/campaign_list_filter_totals';
import {
  CAMPAIGN_LIST_COLUMN_LABELS,
  CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX,
  COLUMN_DRAG_MIME,
  isCampaignListColumnDraggable,
  isCampaignListColumnResizable,
  isCampaignListNumericColumn,
  moveDataColumn,
  resolveCampaignListColumnWidthPx,
  saveCampaignListColumnPrefs,
  type CampaignListColumnId,
  type CampaignListColumnPrefs,
  type CampaignListReorderableColumnId,
  visibleCampaignListColumns,
} from '@/domains/campaigns/list/campaign_list_columns';
import { CampaignListTableBodyRow } from '@/domains/campaigns/list/campaign_list_table_body_row';
import { CampaignListTableHeaderCell } from '@/domains/campaigns/list/campaign_list_table_header_cell';
import { CampaignListTableTotalsCell } from '@/domains/campaigns/list/campaign_list_table_totals_cell';
import type { CampaignSortField, SortOrder } from '@/domains/campaigns/list/campaigns_list_types';
import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/list/campaign_metrics_shared';
import { useCampaignListColumnResize } from '@/domains/campaigns/list/use_campaign_list_column_resize';
import { DirectoryTable, TableBody, TableFooter, TableHeader } from '@/shell/directory_table';
import { cn } from '@/lib/utils';

export type CampaignsListTableProps = {
  items: Campaign[];
  customerNameById: Record<string, string>;
  ownerEmailById: Record<string, string>;
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
  onCampaignOverview?: (campaign: Campaign) => void;
  filterTotals?: CampaignListFilterTotalsView;
  statsQuery?: CampaignStatsQuery;
};

export function CampaignsListTable({
  items,
  customerNameById,
  ownerEmailById,
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
  onCampaignOverview,
  filterTotals,
  statsQuery,
}: CampaignsListTableProps) {
  const columns = visibleCampaignListColumns(columnPrefs);
  const { localWidths, startResize } = useCampaignListColumnResize({
    columnWidths,
    onColumnWidthCommit,
  });
  const columnWidthPxList = columns.map((columnId) =>
    resolveCampaignListColumnWidthPx(columnId, localWidths),
  );
  const tableWidthPx = columnWidthPxList.reduce((sum, widthPx) => sum + widthPx, 0);
  const [draggingColumnId, setDraggingColumnId] = useState<CampaignListReorderableColumnId | null>(
    null,
  );
  const [dragOverColumnId, setDragOverColumnId] = useState<CampaignListReorderableColumnId | null>(
    null,
  );
  const allSelected = items.length > 0 && items.every((item) => selectedIds.has(item.id));
  const totals =
    filterTotals?.totals ??
    sumCampaignListTotals(
      items as CampaignWithMoneyDisplay[],
      metricsById,
      marginsById,
    );
  const funnelTotals =
    filterTotals?.funnelTotals ?? sumCampaignFunnelTotals(items, metricsById);

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
      <div className="admin-campaigns-table-surface p-4">
        <p className="text-muted-foreground">{emptyMessage}</p>
      </div>
    );
  }

  return (
    <DirectoryTable
      className="admin-campaigns-table-surface !rounded-none !border-0 !shadow-none"
      fixedLayout
      tableClassName="admin-table--campaigns"
      tableStyle={{
        width: `${tableWidthPx}px`,
        minWidth: `${tableWidthPx}px`,
        tableLayout: 'fixed',
      }}
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
              const isNum = isCampaignListNumericColumn(columnId);

              return (
                <th
                  key={columnId}
                  className={cn(
                    isSelect ? 'px-4 text-center' : isNum ? 'num' : undefined,
                    columnId === 'name'
                      ? 'campaign-table-th--name campaign-table-cell--tools'
                      : !isSelect
                        ? 'campaign-table-cell--tools'
                        : undefined,
                    draggingColumnId === columnId && 'opacity-60',
                  )}
                >
                  {isSelect ? (
                    <div className="admin-table-cell--select">
                      <Checkbox
                        aria-label="Select all campaigns"
                        checked={allSelected}
                        disabled={fetching}
                        onCheckedChange={(checked) => toggleAll(checked === true)}
                      />
                    </div>
                  ) : (
                    <CampaignListTableHeaderCell
                      appliedOrder={appliedOrder}
                      appliedSort={appliedSort}
                      columnId={columnId}
                      disabled={fetching}
                      dragOver={reorderableTarget != null && dragOverColumnId === reorderableTarget}
                      draggable={draggable}
                      resizable={resizable}
                      resizeLabel={`Resize ${CAMPAIGN_LIST_COLUMN_LABELS[columnId]} column`}
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
                      onResizePointerDown={(event) => {
                        startResize(columnId, event);
                      }}
                    />
                  )}
                </th>
              );
            })}
          </tr>
        </TableHeader>
        <TableBody>
          {items.map((campaign) => (
            <CampaignListTableBodyRow
              key={campaign.id}
              campaign={campaign}
              columns={columns}
              customerNameById={customerNameById}
              fetching={fetching}
              margin={marginsById[campaign.id]}
              metrics={metricsById[campaign.id]}
              ownerEmailById={ownerEmailById}
              selected={selectedIds.has(campaign.id)}
              onCampaignOverview={onCampaignOverview}
              onToggleSelected={toggleOne}
              statsQuery={statsQuery}
            />
          ))}
        </TableBody>
        <TableFooter>
          <tr>
            {columns.map((columnId) => {
              const isNum = isCampaignListNumericColumn(columnId);
              return (
                <td
                  key={columnId}
                  className={cn(
                    columnId === 'select'
                      ? 'px-4 text-center'
                      : columnId === 'name'
                        ? 'campaign-table-td--name campaign-table-cell--tools'
                        : cn('campaign-table-cell--tools', isNum && 'num'),
                  )}
                >
                  <div className="campaign-table-header-cell">
                    <div className="campaign-table-cell__content">
                      <CampaignListTableTotalsCell
                      columnId={columnId}
                      funnelTotals={funnelTotals}
                      pageCount={items.length}
                      totals={totals}
                      totalsLabel={filterTotals ? 'Filtered total' : 'Total'}
                    />
                    </div>
                    {columnId !== 'select' && columnId !== 'name' ? (
                      <div aria-hidden className="campaign-table-body-tools-gutter" />
                    ) : null}
                  </div>
                </td>
              );
            })}
          </tr>
        </TableFooter>
    </DirectoryTable>
  );
}
