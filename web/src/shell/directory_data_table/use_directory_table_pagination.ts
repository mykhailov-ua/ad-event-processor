import { useEffect, useMemo, useState } from 'react';

export type DirectoryTablePaginationState = {
  page: number;
  pageCount: number;
  pageSize: number;
  start: number;
  end: number;
  totalRows: number;
  setPage: (page: number) => void;
};

export function useDirectoryTablePagination(
  totalRows: number,
  pageSize = 10
): DirectoryTablePaginationState {
  const [page, setPage] = useState(0);
  const pageCount = Math.max(1, Math.ceil(totalRows / pageSize));
  const clampedPage = Math.min(page, pageCount - 1);

  useEffect(() => {
    if (page !== clampedPage) {
      setPage(clampedPage);
    }
  }, [clampedPage, page]);

  const start = clampedPage * pageSize;
  const end = Math.min(totalRows, start + pageSize);

  return useMemo(
    () => ({
      page: clampedPage,
      pageCount,
      pageSize,
      start,
      end,
      totalRows,
      setPage,
    }),
    [clampedPage, end, pageCount, pageSize, start, totalRows]
  );
}

export function sliceDirectoryTableRows<Row>(
  rows: readonly Row[],
  pagination: Pick<DirectoryTablePaginationState, 'start' | 'end'>
): Row[] {
  if (rows.length === 0) {
    return [];
  }
  return rows.slice(pagination.start, pagination.end);
}
