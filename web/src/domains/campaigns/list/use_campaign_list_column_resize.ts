import { useCallback, useEffect, useRef, useState, type PointerEvent as ReactPointerEvent } from 'react';

import {
  CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX,
  clampUserResizedCampaignListColumnWidthPx,
  type CampaignListColumnId,
} from '@/domains/campaigns/list/campaign_list_columns';

type ResizeState = {
  columnId: CampaignListColumnId;
  startX: number;
  startWidth: number;
};

function columnWidthsEqual(
  left: Readonly<Partial<Record<CampaignListColumnId, number>>>,
  right: Readonly<Partial<Record<CampaignListColumnId, number>>>,
): boolean {
  const keys = new Set([
    ...Object.keys(left),
    ...Object.keys(right),
  ]) as Set<CampaignListColumnId>;
  for (const key of keys) {
    if (left[key] !== right[key]) {
      return false;
    }
  }
  return true;
}

export function useCampaignListColumnResize({
  columnWidths,
  onColumnWidthCommit,
}: {
  columnWidths: Record<CampaignListColumnId, number>;
  onColumnWidthCommit: (columnId: CampaignListColumnId, widthPx: number) => void;
}) {
  const [localWidths, setLocalWidths] = useState(columnWidths);
  const resizeRef = useRef<ResizeState | null>(null);

  useEffect(() => {
    if (resizeRef.current) {
      return;
    }
    setLocalWidths((current) => (columnWidthsEqual(current, columnWidths) ? current : columnWidths));
  }, [columnWidths]);

  const startResize = useCallback(
    (columnId: CampaignListColumnId, event: ReactPointerEvent<HTMLDivElement>) => {
      event.preventDefault();
      event.stopPropagation();
      const target = event.currentTarget;
      if (target.setPointerCapture) {
        target.setPointerCapture(event.pointerId);
      }
      resizeRef.current = {
        columnId,
        startX: event.clientX,
        startWidth: localWidths[columnId] ?? CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[columnId],
      };
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';
    },
    [localWidths],
  );

  useEffect(() => {
    function finishResize(state: ResizeState) {
      resizeRef.current = null;
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
      setLocalWidths((current) => {
        onColumnWidthCommit(state.columnId, current[state.columnId]);
        return current;
      });
    }

    function onPointerMove(event: PointerEvent) {
      const state = resizeRef.current;
      if (!state) {
        return;
      }
      const delta = event.clientX - state.startX;
      const nextWidth = clampUserResizedCampaignListColumnWidthPx(
        state.columnId,
        state.startWidth + delta,
      );
      setLocalWidths((current) => ({
        ...current,
        [state.columnId]: nextWidth,
      }));
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
  }, [onColumnWidthCommit]);

  return { localWidths, startResize };
}
