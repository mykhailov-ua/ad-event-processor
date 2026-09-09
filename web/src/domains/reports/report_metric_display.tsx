import type { ReactNode } from 'react';

import { displayCount, displayMicro } from '@/lib/display';
import { adminTypography } from '@/lib/admin_spacing';
import { cn } from '@/lib/utils';

export function formatPct(value?: number): string {
  if (value == null || !Number.isFinite(value)) {
    return '-';
  }
  return `${value.toFixed(1)}%`;
}

export function formatRatio(value?: number): string {
  if (value == null || !Number.isFinite(value)) {
    return '-';
  }
  return `${(value * 100).toFixed(2)}%`;
}

function deltaSuffix(delta?: number): string {
  if (delta == null || delta === 0) {
    return '';
  }
  const sign = delta > 0 ? '+' : '';
  return ` (${sign}${displayCount(delta)})`;
}

export function metricCell(
  value: number | undefined,
  delta?: number,
  format: 'count' | 'micro' = 'count'
): ReactNode {
  const text = format === 'micro' ? displayMicro(value) || '-' : displayCount(value) || '-';
  const suffix = deltaSuffix(delta);
  if (!suffix) {
    return text;
  }
  return (
    <span>
      {text}
      <span className={cn('ml-1', adminTypography.captionPlain)}>{suffix}</span>
    </span>
  );
}
