import { useEffect, useRef } from 'react';
import type { ChartCategoryItem } from '../charts/chart_types.js';

export type CategoryPieChartProps = {
  items: ChartCategoryItem[];
  ariaLabel?: string;
  className?: string;
};

/**
 * Donut chart with HTML legend (lazy-loads canvas pie adapter).
 */
export function CategoryPieChart({
  items,
  ariaLabel = 'Donut chart',
  className,
}: CategoryPieChartProps) {
  const hostRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<{ destroy: () => void } | null>(null);

  useEffect(() => {
    if (!hostRef.current || items.length === 0) return undefined;
    let cancelled = false;
    chartRef.current?.destroy();
    chartRef.current = null;
    void import('../charts/pie_chart.js').then((mod) => {
      if (cancelled || !hostRef.current) return;
      chartRef.current = mod.mountPieChart(hostRef.current, items, ariaLabel);
    });
    return () => {
      cancelled = true;
      chartRef.current?.destroy();
      chartRef.current = null;
    };
  }, [items, ariaLabel]);

  if (items.length === 0) {
    return <p className="text-muted text-sm">No data to chart.</p>;
  }

  return <div ref={hostRef} className={className} />;
}
