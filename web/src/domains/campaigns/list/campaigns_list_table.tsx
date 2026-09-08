import { useCallback, useEffect, useMemo, useRef, useState, type DragEvent } from 'react';

import { Checkbox } from '@/components/ui/checkbox';
import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin, CampaignStatsQuery } from '@/api/types';
import { sumCampaignListTotals } from '@/domains/campaigns/list/campaign_list_format';
import { sumCampaignFunnelTotals } from '@/domains/campaigns/list/campaign_list_funnel';
import type { CampaignListFilterTotalsView } from '@/domains/campaigns/list/campaign_list_filter_totals';
import {
  CAMPAIGN_LIST_COLUMN_LABELS,
  CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX,
  COLUMN_DRAG_MIME,
  isCampaignListColumnDraggable,
  isCampaignListColumnResizable,
  isCampaignListNumericColumn,
  isCampaignListPinnedColumn,
  moveDataColumn,
  resolveCampaignListColumnWidthPx,
  type CampaignListColumnId,
  type CampaignListColumnPrefs,
  type CampaignListReorderableColumnId,
  visibleCampaignListColumns,
} from '@/domains/campaigns/list/campaign_list_columns';
import {
  campaignListPinnedCellClassName,
  campaignListPinnedColumnStyle,
  campaignListPinnedEdgeWidthPx,
  syncDirectoryPinnedEdgeShadow,
} from '@/domains/campaigns/list/campaign_list_pinned_columns';
import { buildCampaignRowVmCache } from '@/domains/campaigns/list/campaign_list_row_vm';
import { CampaignListTableBodyRow } from '@/domains/campaigns/list/campaign_list_table_body_row';
import {
  CampaignListColumnResizeHandle,
  CampaignListTableHeaderCell,
} from '@/domains/campaigns/list/campaign_list_table_header_cell';
import { CampaignListTableTotalsCell } from '@/domains/campaigns/list/campaign_list_table_totals_cell';
import type { CampaignSortField, SortOrder } from '@/domains/campaigns/list/campaigns_list_types';
import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/list/campaign_metrics_shared';
import { useCampaignListColumnResize } from '@/domains/campaigns/list/use_campaign_list_column_resize';
import {
  campaignListCellContentClass,
  campaignListCellContentNumClass,
  campaignListSelectHeaderShellClass,
  campaignListTableClass,
  campaignListTdClass,
  campaignListTfootTdClass,
  campaignListThClass,
} from '@/domains/campaigns/list/campaign_list_classes';
import {
  DirectoryTable,
  TableBody,
  TableFooter,
  TableHeader,
  directoryTableRevalidatingClass,
} from '@/shell/directory_table';
import { cn } from '@/lib/utils';

export type CampaignsListTableProps = {
  items?: Campaign[];
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
  listRevalidating?: boolean;
  emptyMessage?: string;
  onCampaignOverview?: (campaign: Campaign) => void;
  filterTotals?: CampaignListFilterTotalsView;
  statsCacheRevision: string;
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
  listRevalidating = false,
  emptyMessage = 'No campaigns match the current filters.',
  onCampaignOverview,
  filterTotals,
  statsCacheRevision,
  statsQuery,
}: CampaignsListTableProps) {
  const columns = visibleCampaignListColumns(columnPrefs);
  const rowVmCache = useMemo(
    () =>
      buildCampaignRowVmCache(
        items ?? [],
        metricsById,
        marginsById,
        customerNameById,
        ownerEmailById
      ),
    [customerNameById, items, marginsById, metricsById, ownerEmailById]
  );
  const tableRef = useRef<HTMLTableElement>(null);
  const colgroupRef = useRef<HTMLTableColElement>(null);
  const hostRef = useRef<HTMLDivElement>(null);
  const { startResize } = useCampaignListColumnResize({
    columnWidths,
    columns,
    colgroupRef,
    hostRef,
    onColumnWidthCommit,
    tableRef,
  });
  const columnWidthPxList = columns.map((columnId) =>
    resolveCampaignListColumnWidthPx(columnId, columnWidths)
  );
  const tableWidthPx = columnWidthPxList.reduce((sum, widthPx) => sum + widthPx, 0);
  const pinnedEdgeWidthPx = useMemo(
    () => campaignListPinnedEdgeWidthPx(columns, columnWidths),
    [columns, columnWidths]
  );

  useEffect(() => {
    syncDirectoryPinnedEdgeShadow(hostRef.current, columns, columnWidths);
  }, [columns, columnWidths, pinnedEdgeWidthPx]);

  const [draggingColumnId, setDraggingColumnId] = useState<CampaignListReorderableColumnId | null>(
    null
  );
  const allSelected =
    (items ?? []).length > 0 && (items ?? []).every((item) => selectedIds.has(item.id));
  const totals =
    filterTotals?.totals ??
    sumCampaignListTotals((items ?? []) as CampaignWithMoneyDisplay[], metricsById, marginsById);
  const funnelTotals =
    filterTotals?.funnelTotals ?? sumCampaignFunnelTotals(items ?? [], metricsById);

  function toggleAll(checked: boolean) {
    if (!checked) {
      onSelectedIdsChange(new Set());
      return;
    }
    onSelectedIdsChange(new Set((items ?? []).map((item) => item.id)));
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
    (targetId: CampaignListReorderableColumnId, event: DragEvent) => {
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
      setDraggingColumnId(null);
    },
    [columnPrefs, onColumnPrefsChange]
  );

  const clearDragState = useCallback(() => {
    setDraggingColumnId(null);
  }, []);

  if ((items ?? []).length === 0) {
    return (
      <div className="p-4">
        <p className="text-muted-foreground">{emptyMessage}</p>
      </div>
    );
  }

  return (
    <DirectoryTable
      className={cn(
        'rounded-none border-0 shadow-none',
        directoryTableRevalidatingClass(listRevalidating)
      )}
      fixedLayout
      hostRef={hostRef}
      horizontalScroll
      nested
      pinnedEdgeWidthPx={pinnedEdgeWidthPx}
      tableClassName={campaignListTableClass}
      tableRef={tableRef}
      tableStyle={{
        width: `${tableWidthPx}px`,
        minWidth: `${tableWidthPx}px`,
        tableLayout: 'fixed',
      }}
    >
      <colgroup ref={colgroupRef}>
        {columns.map((columnId, index) => {
          const widthPx = columnWidthPxList[index] ?? CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[columnId];
          return <col key={columnId} style={{ width: `${widthPx}px` }} />;
        })}
      </colgroup>
      <TableHeader>
        <tr>
          {columns.map((columnId) => {
            const draggable = isCampaignListColumnDraggable(columnId);
            const resizable = isCampaignListColumnResizable(columnId, columns);
            const reorderableTarget = draggable ? columnId : null;
            const isSelect = columnId === 'select';

            return (
              <th
                key={columnId}
                className={cn(
                  campaignListThClass,
                  isSelect ? 'text-center' : undefined,
                  isCampaignListPinnedColumn(columnId) &&
                    campaignListPinnedCellClassName(columnId, columns, 'header'),
                  draggingColumnId === columnId && 'opacity-60'
                )}
                data-col-pin={isCampaignListPinnedColumn(columnId) ? columnId : undefined}
                style={campaignListPinnedColumnStyle(columnId, columns, columnWidths)}
              >
                {isSelect ? (
                  <div className={campaignListSelectHeaderShellClass}>
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
                    draggable={draggable}
                    onColumnSort={onColumnSort}
                    onDragEnd={clearDragState}
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
                  <CampaignListColumnResizeHandle
                    label={`Resize ${CAMPAIGN_LIST_COLUMN_LABELS[columnId]} column`}
                    onPointerDown={(event) => {
                      startResize(columnId, event);
                    }}
                  />
                ) : null}
              </th>
            );
          })}
        </tr>
      </TableHeader>
      <TableBody>
        {(items ?? []).map((campaign) => (
          <CampaignListTableBodyRow
            key={campaign.id}
            campaign={campaign}
            columnWidths={columnWidths}
            columns={columns}
            customerNameById={customerNameById}
            fetching={fetching}
            margin={marginsById[campaign.id]}
            metrics={metricsById[campaign.id]}
            ownerEmailById={ownerEmailById}
            selected={selectedIds.has(campaign.id)}
            onCampaignOverview={onCampaignOverview}
            onToggleSelected={toggleOne}
            rowVmCache={rowVmCache}
            statsCacheRevision={statsCacheRevision}
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
                  campaignListTdClass,
                  campaignListTfootTdClass,
                  columnId === 'select' || columnId === 'id' ? 'p-0' : undefined,
                  isCampaignListPinnedColumn(columnId) &&
                    campaignListPinnedCellClassName(columnId, columns, 'footer')
                )}
                data-col-pin={isCampaignListPinnedColumn(columnId) ? columnId : undefined}
                style={campaignListPinnedColumnStyle(columnId, columns, columnWidths)}
              >
                <div
                  className={isNum ? campaignListCellContentNumClass : campaignListCellContentClass}
                >
                  <CampaignListTableTotalsCell
                    columnId={columnId}
                    funnelTotals={funnelTotals}
                    pageCount={(items ?? []).length}
                    totals={totals}
                    totalsLabel={filterTotals ? 'Filtered total' : 'Total'}
                  />
                </div>
              </td>
            );
          })}
        </tr>
      </TableFooter>
    </DirectoryTable>
  );
}
