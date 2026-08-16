import { useEffect, useRef } from 'react';
import type { HourlyMetricRow } from '../helpers/chart_pool.js';
import type { SpendCurvePoint } from '../charts/campaign_chart_types.js';

export type { SpendCurvePoint } from '../charts/campaign_chart_types.js';

export type CampaignHourlyChartProps = {
  hourly?: HourlyMetricRow[] | null;
  field?: string;
  label?: string;
  className?: string;
};

/**
 * uPlot time-series for campaign hourly stats (lazy-loaded).
 */
export function CampaignHourlyChart({
  hourly,
  field,
  label,
  className,
}: CampaignHourlyChartProps) {
  const mountRef = useRef<HTMLDivElement>(null);
  const handleRef = useRef<{ destroy: () => void } | null>(null);

  useEffect(() => {
    const mount = mountRef.current;
    if (!mount) return undefined;
    let cancelled = false;
    handleRef.current?.destroy();
    handleRef.current = null;
    void import('../charts/campaign_series_uplot.js').then((mod) => {
      if (cancelled || !mountRef.current) return;
      handleRef.current = mod.mountCampaignSeriesChart(mountRef.current, hourly, { field, label });
    });
    return () => {
      cancelled = true;
      handleRef.current?.destroy();
      handleRef.current = null;
    };
  }, [hourly, field, label]);

  return <div ref={mountRef} className={className} />;
}

export type CampaignSpendCurveChartProps = {
  curve?: SpendCurvePoint[] | null;
  field?: 'impressions' | 'spend_micro';
  className?: string;
};

/**
 * Forecast spend-curve line chart (uPlot).
 */
export function CampaignSpendCurveChart({
  curve,
  field = 'impressions',
  className,
}: CampaignSpendCurveChartProps) {
  const mountRef = useRef<HTMLDivElement>(null);
  const handleRef = useRef<{ destroy: () => void } | null>(null);

  useEffect(() => {
    const mount = mountRef.current;
    if (!mount) return undefined;
    let cancelled = false;
    handleRef.current?.destroy();
    handleRef.current = null;
    void import('../charts/campaign_series_uplot.js').then((mod) => {
      if (cancelled || !mountRef.current) return;
      handleRef.current = mod.mountSpendCurveChart(mountRef.current, curve, field);
    });
    return () => {
      cancelled = true;
      handleRef.current?.destroy();
      handleRef.current = null;
    };
  }, [curve, field]);

  return <div ref={mountRef} className={className} />;
}
