import { useCallback, useEffect, useRef, type PointerEvent as ReactPointerEvent, type RefObject } from 'react';

type ResizeState<T extends string> = {
  columnId: T;
  columnIndex: number;
  startX: number;
  startWidth: number;
};

function sumColumnWidthsPx<T extends string>(
  columns: readonly T[],
  widths: Readonly<Record<T, number>>,
  minWidthPx: (columnId: T) => number,
  override?: { columnId: T; widthPx: number },
): number {
  return columns.reduce((sum, columnId) => {
    const widthPx =
      override?.columnId === columnId ? override.widthPx : widths[columnId] ?? minWidthPx(columnId);
    return sum + widthPx;
  }, 0);
}

function applyTableColumnWidths<T extends string>(
  colgroup: HTMLTableColElement,
  columns: readonly T[],
  widths: Readonly<Record<T, number>>,
  minWidthPx: (columnId: T) => number,
  override?: { columnId: T; widthPx: number },
): number {
  const totalWidthPx = sumColumnWidthsPx(columns, widths, minWidthPx, override);
  columns.forEach((columnId, index) => {
    const col = colgroup.children.item(index) as HTMLTableColElement | null;
    if (!col) {
      return;
    }
    const widthPx =
      override?.columnId === columnId ? override.widthPx : widths[columnId] ?? minWidthPx(columnId);
    col.style.width = `${widthPx}px`;
  });
  return totalWidthPx;
}

export function useDirectoryColumnResize<T extends string>({
  columnWidths,
  columns,
  minWidthPx,
  clampUserWidth,
  onColumnWidthCommit,
  colgroupRef,
  tableRef,
}: {
  columnWidths: Record<T, number>;
  columns: readonly T[];
  minWidthPx: (columnId: T) => number;
  clampUserWidth: (columnId: T, widthPx: number) => number;
  onColumnWidthCommit: (columnId: T, widthPx: number) => void;
  colgroupRef: RefObject<HTMLTableColElement | null>;
  tableRef: RefObject<HTMLTableElement | null>;
}) {
  const resizeRef = useRef<ResizeState<T> | null>(null);
  const draftWidthRef = useRef<number | null>(null);

  const startResize = useCallback(
    (columnId: T, event: ReactPointerEvent<HTMLDivElement>) => {
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
      const startWidth = columnWidths[columnId] ?? minWidthPx(columnId);
      resizeRef.current = {
        columnId,
        columnIndex,
        startX: event.clientX,
        startWidth,
      };
      draftWidthRef.current = startWidth;
    },
    [columnWidths, columns, minWidthPx],
  );

  useEffect(() => {
    function commitResize() {
      const state = resizeRef.current;
      const colgroup = colgroupRef.current;
      const table = tableRef.current;
      if (!state || !colgroup || !table) {
        return;
      }
      const widthPx = clampUserWidth(
        state.columnId,
        draftWidthRef.current ??
          columnWidths[state.columnId] ??
          minWidthPx(state.columnId),
      );
      onColumnWidthCommit(state.columnId, widthPx);
      resizeRef.current = null;
      draftWidthRef.current = null;
    }

    function onPointerMove(event: PointerEvent) {
      const state = resizeRef.current;
      const colgroup = colgroupRef.current;
      const table = tableRef.current;
      if (!state || !colgroup || !table) {
        return;
      }
      const delta = event.clientX - state.startX;
      const nextWidth = clampUserWidth(state.columnId, state.startWidth + delta);
      draftWidthRef.current = nextWidth;
      const totalWidthPx = applyTableColumnWidths(colgroup, columns, columnWidths, minWidthPx, {
        columnId: state.columnId,
        widthPx: nextWidth,
      });
      table.style.width = `${totalWidthPx}px`;
      table.style.minWidth = `${totalWidthPx}px`;
    }

    function onPointerUp() {
      if (!resizeRef.current) {
        return;
      }
      commitResize();
    }

    window.addEventListener('pointermove', onPointerMove);
    window.addEventListener('pointerup', onPointerUp);
    window.addEventListener('pointercancel', onPointerUp);
    return () => {
      window.removeEventListener('pointermove', onPointerMove);
      window.removeEventListener('pointerup', onPointerUp);
      window.removeEventListener('pointercancel', onPointerUp);
    };
  }, [clampUserWidth, colgroupRef, columnWidths, columns, minWidthPx, onColumnWidthCommit, tableRef]);

  return { startResize };
}
