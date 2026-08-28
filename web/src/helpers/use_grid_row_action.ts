import { useCallback, useRef, type ChangeEvent, type KeyboardEvent, type MouseEvent } from 'react';

export function useGridRowAction(
  handler: (rowId: string) => void
): (event: MouseEvent<HTMLElement>) => void {
  const handlerRef = useRef(handler);
  handlerRef.current = handler;
  return useCallback((event: MouseEvent<HTMLElement>) => {
    const rowId = event.currentTarget.dataset.rowId;
    if (!rowId) return;
    handlerRef.current(rowId);
  }, []);
}

export function useGridRowActionPair(
  handler: (rowId: string, source: string) => void
): (event: MouseEvent<HTMLElement>) => void {
  const handlerRef = useRef(handler);
  handlerRef.current = handler;
  return useCallback((event: MouseEvent<HTMLElement>) => {
    const rowId = event.currentTarget.dataset.rowId;
    const source = event.currentTarget.dataset.rowSource;
    if (!rowId || !source) return;
    handlerRef.current(rowId, source);
  }, []);
}

export function useGridRowDatasetAction(
  handler: (rowId: string, action: string) => void
): (event: MouseEvent<HTMLElement>) => void {
  const handlerRef = useRef(handler);
  handlerRef.current = handler;
  return useCallback((event: MouseEvent<HTMLElement>) => {
    const rowId = event.currentTarget.dataset.rowId;
    const action = event.currentTarget.dataset.rowAction;
    if (!rowId || !action) return;
    handlerRef.current(rowId, action);
  }, []);
}

export function useGridRowCheckboxChange(
  handler: (rowId: string, checked: boolean) => void
): (event: ChangeEvent<HTMLInputElement>) => void {
  const handlerRef = useRef(handler);
  handlerRef.current = handler;
  return useCallback((event: ChangeEvent<HTMLInputElement>) => {
    const rowId = event.currentTarget.dataset.rowId;
    if (!rowId) return;
    handlerRef.current(rowId, event.target.checked);
  }, []);
}

export function useGridRowActivate(handler: (rowId: string) => void): {
  onClick: (event: MouseEvent<HTMLElement>) => void;
  onKeyDown: (event: KeyboardEvent<HTMLElement>) => void;
} {
  const handlerRef = useRef(handler);
  handlerRef.current = handler;
  const activate = useCallback((target: HTMLElement) => {
    const rowId = target.dataset.rowId;
    if (rowId) handlerRef.current(rowId);
  }, []);
  const onClick = useCallback(
    (event: MouseEvent<HTMLElement>) => {
      activate(event.currentTarget);
    },
    [activate]
  );
  const onKeyDown = useCallback(
    (event: KeyboardEvent<HTMLElement>) => {
      if (event.key !== 'Enter' && event.key !== ' ') return;
      event.preventDefault();
      activate(event.currentTarget);
    },
    [activate]
  );
  return { onClick, onKeyDown };
}
