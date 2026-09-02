import { useCallback, useEffect, useRef, useState } from 'react';

import {
  CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX,
  type CampaignListColumnId,
} from '@/domains/campaigns/campaign_list_columns';

type ResizeState = {
  columnId: CampaignListColumnId;
  startX: number;
  startWidth: number;
};

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
    if (!resizeRef.current) {
      setLocalWidths(columnWidths);
    }
  }, [columnWidths]);

  const startResize = useCallback(
    (columnId: CampaignListColumnId, clientX: number) => {
      resizeRef.current = {
        columnId,
        startX: clientX,
        startWidth: localWidths[columnId],
      };
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';
    },
    [localWidths],
  );

  useEffect(() => {
    function onPointerMove(event: PointerEvent) {
      const state = resizeRef.current;
      if (!state) {
        return;
      }
      const delta = event.clientX - state.startX;
      const nextWidth = Math.max(
        CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[state.columnId],
        state.startWidth + delta,
      );
      setLocalWidths((current) => ({
        ...current,
        [state.columnId]: nextWidth,
      }));
    }

    function onPointerUp() {
      const state = resizeRef.current;
      if (!state) {
        return;
      }
      resizeRef.current = null;
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
      setLocalWidths((current) => {
        onColumnWidthCommit(state.columnId, current[state.columnId]);
        return current;
      });
    }

    window.addEventListener('pointermove', onPointerMove);
    window.addEventListener('pointerup', onPointerUp);
    return () => {
      window.removeEventListener('pointermove', onPointerMove);
      window.removeEventListener('pointerup', onPointerUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };
  }, [onColumnWidthCommit]);

  return { localWidths, startResize };
}
