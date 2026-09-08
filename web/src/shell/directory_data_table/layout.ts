export function directoryTableWidthPx<T extends string>(
  columns: readonly T[],
  widths: Readonly<Record<T, number>>
): number {
  return columns.reduce((sum, columnId) => sum + (widths[columnId] ?? 0), 0);
}

/** @deprecated Prefer proportionalDirectoryColumnWidths. */
export function stretchDirectoryColumnWidths<T extends string>(
  columns: readonly T[],
  baseWidths: Readonly<Record<T, number>>,
  containerWidthPx: number,
  stretchColumnId: T
): Record<T, number> {
  const total = directoryTableWidthPx(columns, baseWidths);
  if (containerWidthPx <= 0 || total >= containerWidthPx) {
    return { ...baseWidths };
  }
  const stretched = { ...baseWidths } as Record<T, number>;
  stretched[stretchColumnId] = (baseWidths[stretchColumnId] ?? 0) + (containerWidthPx - total);
  return stretched;
}

export function proportionalDirectoryColumnWidths<T extends string>(
  columns: readonly T[],
  baseWidths: Readonly<Record<T, number>>,
  containerWidthPx: number,
  maxWidths: Partial<Record<T, number>> = {}
): Record<T, number> {
  const baseTotal = directoryTableWidthPx(columns, baseWidths);
  if (containerWidthPx <= 0 || baseTotal <= 0 || baseTotal >= containerWidthPx) {
    return { ...baseWidths };
  }

  const stretched = { ...baseWidths } as Record<T, number>;
  let remaining = containerWidthPx - baseTotal;
  let activeColumns = columns.filter((columnId) => {
    const max = maxWidths[columnId];
    return max == null || (baseWidths[columnId] ?? 0) < max;
  });

  while (remaining > 0 && activeColumns.length > 0) {
    const activeTotal = activeColumns.reduce((sum, columnId) => sum + stretched[columnId], 0);
    if (activeTotal <= 0) {
      break;
    }

    let distributed = 0;
    const nextActive: T[] = [];

    for (const columnId of activeColumns) {
      const share = Math.floor((remaining * stretched[columnId]) / activeTotal);
      if (share <= 0) {
        nextActive.push(columnId);
        continue;
      }
      const max = maxWidths[columnId];
      const candidate = stretched[columnId] + share;
      if (max != null && candidate >= max) {
        const added = max - stretched[columnId];
        stretched[columnId] = max;
        distributed += added;
      } else {
        stretched[columnId] = candidate;
        distributed += share;
        nextActive.push(columnId);
      }
    }

    if (distributed === 0) {
      for (const columnId of activeColumns) {
        const max = maxWidths[columnId];
        if (max != null && stretched[columnId] >= max) {
          continue;
        }
        stretched[columnId] += 1;
        distributed += 1;
        if (max == null || stretched[columnId] < max) {
          nextActive.push(columnId);
        }
        break;
      }
    }

    remaining -= distributed;
    activeColumns = nextActive;
  }

  return stretched;
}

export function resolveDashboardBreakdownLayoutMaxWidths(
  containerWidthPx: number
): Partial<Record<string, number>> {
  if (containerWidthPx <= 0) {
    return {};
  }
  const nameCap = Math.min(300, Math.round(containerWidthPx * 0.28));
  return {
    name: nameCap,
    unique_clicks: 260,
    conversions: 148,
    cost: 140,
    revenue: 140,
    profit: 160,
    cpc: 116,
    cpa: 116,
    cr: 96,
    epc: 116,
    roi: 144,
  };
}

export function resolveDashboardRecentClickLayoutMaxWidths(
  containerWidthPx: number
): Partial<Record<string, number>> {
  if (containerWidthPx <= 0) {
    return {};
  }
  const idCap = Math.min(300, Math.round(containerWidthPx * 0.3));
  return {
    click_id: idCap,
    campaign_id: idCap,
    created_at: 200,
    country: 88,
    sub1: 200,
    placement_id: 200,
    goal_name: 200,
    cost: 128,
    revenue: 128,
  };
}
