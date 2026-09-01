export type ListPageRange = {
  page: number;
  pageCount: number;
  rangeStart: number;
  rangeEnd: number;
};

export function listPageRange(
  total: number,
  limit: number,
  offset: number,
  itemCount: number,
): ListPageRange {
  if (total <= 0 || limit <= 0) {
    return { page: 0, pageCount: 0, rangeStart: 0, rangeEnd: 0 };
  }

  const page = Math.floor(offset / limit) + 1;
  const pageCount = Math.max(1, Math.ceil(total / limit));
  const rangeStart = offset + 1;
  const rangeEnd = Math.min(offset + itemCount, total);

  return { page, pageCount, rangeStart, rangeEnd };
}
