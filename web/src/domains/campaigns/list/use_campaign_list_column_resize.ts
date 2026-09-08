// campaign list column resize: pointer capture on handle; colgroup/table width updated during drag; commit on pointerup only.
import {
  useCallback,
  useEffect,
  useRef,
  type PointerEvent as ReactPointerEvent,
  type RefObject,
} from 'react';

import {
  CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX,
  clampUserResizedCampaignListColumnWidthPx,
  type CampaignListColumnId,
} from '@/domains/campaigns/list/campaign_list_columns';
import { syncDirectoryPinnedEdgeShadow } from '@/domains/campaigns/list/campaign_list_pinned_columns';

type ResizeState = {
  columnId: CampaignListColumnId;
  columnIndex: number;
  startX: number;
  startWidth: number;
};

function sumColumnWidthsPx(
  columns: CampaignListColumnId[],
  widths: Readonly<Record<CampaignListColumnId, number>>,
  override?: { columnId: CampaignListColumnId; widthPx: number }
): number {
  return columns.reduce((sum, columnId) => {
    const widthPx =
      override?.columnId === columnId
        ? override.widthPx
        : (widths[columnId] ?? CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[columnId]);
    return sum + widthPx;
  }, 0);
}

function applyTableColumnWidths(
  colgroup: HTMLTableColElement,
  columns: CampaignListColumnId[],
  widths: Readonly<Record<CampaignListColumnId, number>>,
  override?: { columnId: CampaignListColumnId; widthPx: number }
): number {
  const totalWidthPx = sumColumnWidthsPx(columns, widths, override);
  columns.forEach((columnId, index) => {
    const col = colgroup.children.item(index) as HTMLTableColElement | null;
    if (!col) {
      return;
    }
    const widthPx =
      override?.columnId === columnId
        ? override.widthPx
        : (widths[columnId] ?? CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[columnId]);
    col.style.width = `${widthPx}px`;
  });
  return totalWidthPx;
}

export function useCampaignListColumnResize({
  columnWidths,
  columns,
  onColumnWidthCommit,
  colgroupRef,
  tableRef,
  hostRef,
}: {
  columnWidths: Record<CampaignListColumnId, number>;
  columns: CampaignListColumnId[];
  onColumnWidthCommit: (columnId: CampaignListColumnId, widthPx: number) => void;
  colgroupRef: RefObject<HTMLTableColElement | null>;
  tableRef: RefObject<HTMLTableElement | null>;
  hostRef?: RefObject<HTMLDivElement | null>;
}) {
  const resizeRef = useRef<ResizeState | null>(null);
  const draftWidthRef = useRef<number | null>(null);

  const startResize = useCallback(
    (columnId: CampaignListColumnId, event: ReactPointerEvent<HTMLDivElement>) => {
      event.preventDefault();
      event.stopPropagation();
      const columnIndex = columns.indexOf(columnId);
      if (columnIndex < 0) {
        return;
      }
      const target = event.currentTarget;
      if (target.setPointerCapture) {
        target.setPointerCapture(event.pointerId);
      }
      const startWidth = columnWidths[columnId] ?? CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[columnId];
      resizeRef.current = {
        columnId,
        columnIndex,
        startX: event.clientX,
        startWidth,
      };
      draftWidthRef.current = startWidth;
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';
    },
    [columnWidths, columns]
  );

  useEffect(() => {
    function applyDraftWidth(state: ResizeState, widthPx: number) {
      const colgroup = colgroupRef.current;
      const table = tableRef.current;
      if (!colgroup) {
        return;
      }
      const totalWidthPx = applyTableColumnWidths(colgroup, columns, columnWidths, {
        columnId: state.columnId,
        widthPx,
      });
      if (table) {
        table.style.width = `${totalWidthPx}px`;
        table.style.minWidth = `${totalWidthPx}px`;
      }
      syncDirectoryPinnedEdgeShadow(hostRef?.current, columns, columnWidths, {
        columnId: state.columnId,
        widthPx,
      });
    }

    function finishResize(state: ResizeState) {
      const widthPx =
        draftWidthRef.current ??
        columnWidths[state.columnId] ??
        CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[state.columnId];
      resizeRef.current = null;
      draftWidthRef.current = null;
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
      onColumnWidthCommit(state.columnId, widthPx);
    }

    function onPointerMove(event: PointerEvent) {
      const state = resizeRef.current;
      if (!state) {
        return;
      }
      const delta = event.clientX - state.startX;
      const nextWidth = clampUserResizedCampaignListColumnWidthPx(
        state.columnId,
        state.startWidth + delta
      );
      draftWidthRef.current = nextWidth;
      applyDraftWidth(state, nextWidth);
    }

    function onPointerUp(event: PointerEvent) {
      const state = resizeRef.current;
      if (!state) {
        return;
      }
      finishResize(state);
      if (event.currentTarget instanceof HTMLElement && event.currentTarget.releasePointerCapture) {
        try {
          event.currentTarget.releasePointerCapture(event.pointerId);
        } catch {
          // Grip may already have lost capture.
        }
      }
    }

    function onPointerCancel() {
      const state = resizeRef.current;
      if (!state) {
        return;
      }
      finishResize(state);
    }

    window.addEventListener('pointermove', onPointerMove);
    window.addEventListener('pointerup', onPointerUp);
    window.addEventListener('pointercancel', onPointerCancel);
    return () => {
      window.removeEventListener('pointermove', onPointerMove);
      window.removeEventListener('pointerup', onPointerUp);
      window.removeEventListener('pointercancel', onPointerCancel);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };
  }, [colgroupRef, columnWidths, columns, hostRef, onColumnWidthCommit, tableRef]);

  return { startResize };
}
