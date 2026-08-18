import { useEffect, useRef } from 'react';
import type { DashboardSummary } from '../types/index.js';
import type { EdgePanelData, XDPPanelData } from './edge_panel.js';
import { displayLabel } from '../helpers/display_labels.js';
import {
  formatChartTick,
  formatClockTime,
  formatRefreshCountdown,
} from '../helpers/chart_format.js';
import {
  metricColorToken,
  OPS_CHART_RANGE_OPTIONS,
  rangeMsFromHours,
  snapshotSeries,
  type MetricPoint,
} from '../helpers/ops_metric_series.js';
import type { MetricChartHandle } from '../charts/metric_chart.js';
import { SegmentedControl } from './segmented_control.js';

export type OperatorDashCharts = {
  edge?: EdgePanelData;
  xdp?: XDPPanelData;
};

export type OpsMetricSpec = {
  id: string;
  title: string;
  value: number;
  points: MetricPoint[];
  color: string;
  max?: number;
  formatValue?: (value: number) => string;
  displayValue?: string;
};

export function buildOpsMetricSpecs(
  summary: DashboardSummary | null,
  operatorDash: OperatorDashCharts | null,
  metricSeries: Record<string, MetricPoint[]>,
  chartsRangeHours: number
): OpsMetricSpec[] {
  const specs: OpsMetricSpec[] = [];
  let dropIndex = 0;
  const rangeMs = rangeMsFromHours(chartsRangeHours);

  if (summary) {
    const outboxVal = Number(summary.outbox_pending) || 0;
    specs.push({
      id: 'outbox-pending',
      title: 'Outbox pending',
      value: outboxVal,
      points:
        metricSeries['outbox-pending'] ?? snapshotSeries('outbox-pending', outboxVal, rangeMs),
      color: metricColorToken('outbox-pending'),
      displayValue: formatChartTick(outboxVal),
    });
    const rpsVal = Number(summary.rps_estimate) || 0;
    specs.push({
      id: 'rps-estimate',
      title: 'RPS estimate',
      value: rpsVal,
      points: metricSeries['rps-estimate'] ?? snapshotSeries('rps-estimate', rpsVal, rangeMs),
      color: metricColorToken('rps-estimate'),
      formatValue: (v) => v.toFixed(1),
      displayValue: rpsVal.toFixed(1),
    });
    const driftVal = Number(summary.drift_micro_max) || 0;
    specs.push({
      id: 'drift-alert',
      title: 'Drift (micro)',
      value: driftVal,
      points: metricSeries['drift-alert'] ?? snapshotSeries('drift-alert', driftVal, rangeMs),
      color: metricColorToken('drift-alert'),
      formatValue: (v) => formatChartTick(v),
      displayValue: formatChartTick(driftVal),
    });
    const breakerVal = String(summary.emergency_breaker).toLowerCase() === 'open' ? 1 : 0;
    specs.push({
      id: 'emergency-breaker',
      title: 'Emergency breaker',
      value: breakerVal,
      points: snapshotSeries('emergency-breaker', breakerVal, rangeMs),
      color: metricColorToken('emergency-breaker'),
      max: 1,
      formatValue: (v) => (v > 0 ? 'Open' : 'Closed'),
      displayValue: breakerVal > 0 ? 'Open' : 'Closed',
    });
  }

  const edge = operatorDash?.edge;
  if (edge) {
    const ingress = [
      { id: 'ingress-h1', title: 'HTTP/1 ingress', value: Number(edge.ingress_h1) || 0 },
      { id: 'ingress-h2', title: 'HTTP/2 ingress', value: Number(edge.ingress_h2) || 0 },
      { id: 'ingress-h3', title: 'HTTP/3 ingress', value: Number(edge.ingress_h3) || 0 },
    ];
    for (let i = 0; i < ingress.length; i++) {
      const item = ingress[i];
      specs.push({
        ...item,
        points: snapshotSeries(item.id, item.value, rangeMs),
        color: metricColorToken(item.id),
        displayValue: formatChartTick(item.value),
      });
    }
    const botSignals = [
      { id: 'edge-tarpit', title: 'Edge tarpit', value: Number(edge.tarpit_total) || 0 },
      {
        id: 'edge-blacklist-stale',
        title: 'Blacklist stale',
        value: Number(edge.blacklist_stale) || 0,
      },
      {
        id: 'edge-fraud-tier',
        title: 'Fraud tier blocks',
        value: Number(edge.blocked?.fraud_tier) || 0,
      },
    ];
    for (let i = 0; i < botSignals.length; i++) {
      const item = botSignals[i];
      specs.push({
        ...item,
        points: snapshotSeries(item.id, item.value, rangeMs),
        color: metricColorToken(item.id),
        displayValue: formatChartTick(item.value),
      });
    }
  }

  const drops = operatorDash?.xdp?.drops;
  if (drops && typeof drops === 'object') {
    for (const key of Object.keys(drops).sort()) {
      const id = `drop-${key}`;
      const value = Number(drops[key]) || 0;
      specs.push({
        id,
        title: displayLabel(key),
        value,
        points: snapshotSeries(id, value, rangeMs),
        color: metricColorToken(id, dropIndex),
        displayValue: formatChartTick(value),
      });
      dropIndex += 1;
    }
  }

  return specs;
}

function MetricChartCard({ spec, rangeHours }: { spec: OpsMetricSpec; rangeHours: number }) {
  const mountRef = useRef<HTMLDivElement>(null);
  const handleRef = useRef<MetricChartHandle | null>(null);

  useEffect(() => {
    const el = mountRef.current;
    if (!el) return undefined;
    let cancelled = false;
    void import('../charts/metric_chart.js').then((mod) => {
      if (cancelled || !mountRef.current) return;
      const payload = {
        title: spec.title,
        points: spec.points,
        value: spec.value,
        max: spec.max,
        color: spec.color,
        rangeHours,
        formatValue: spec.formatValue,
      };
      if (handleRef.current?.update) {
        handleRef.current.update(payload);
      } else {
        handleRef.current = mod.mountMetricChart(mountRef.current, payload);
      }
    });
    return () => {
      cancelled = true;
    };
  }, [spec.title, spec.points, spec.value, spec.max, spec.color, spec.formatValue, rangeHours]);

  useEffect(
    () => () => {
      handleRef.current?.destroy?.();
      handleRef.current = null;
    },
    []
  );

  return (
    <article className="metric-chart-card">
      <div className="metric-chart-card__head">
        <h3 className="metric-chart-card__title">{spec.title}</h3>
        <span className="metric-chart-card__value" style={{ color: `var(${spec.color})` }}>
          {spec.displayValue ?? formatChartTick(spec.value)}
        </span>
      </div>
      <div ref={mountRef} className="metric-chart-mount" data-chart-id={spec.id} />
    </article>
  );
}

export type OpsMetricChartsProps = {
  specs: OpsMetricSpec[];
  chartsLayout: string;
  chartsRangeHours: number;
  lastUpdatedAt: number;
  nextRefreshAt: number;
  feedMode: string;
  onLayoutChange: (layout: string) => void;
  onRangeChange: (hours: number) => void;
};

export function OpsMetricCharts({
  specs,
  chartsLayout,
  chartsRangeHours,
  lastUpdatedAt,
  nextRefreshAt,
  feedMode,
  onLayoutChange,
  onRangeChange,
}: OpsMetricChartsProps) {
  const now = Date.now();
  const live = feedMode === 'stream';

  if (specs.length === 0) return null;

  return (
    <section className="stack">
      <h2 className="subsection-title">Operations metrics</h2>
      <div className="ops-charts-toolbar">
        <div className="ops-charts-toolbar__controls">
          <div className="ops-charts-toolbar__group">
            <p className="ops-charts-toolbar__label">Range</p>
            <SegmentedControl
              items={OPS_CHART_RANGE_OPTIONS.map((opt) => ({
                value: String(opt.value),
                label: opt.label,
              }))}
              selected={String(chartsRangeHours)}
              onChange={(value) => {
                const hours = Number(value);
                if (hours === 1 || hours === 6 || hours === 12 || hours === 24) {
                  onRangeChange(hours);
                }
              }}
            />
          </div>
          <div className="ops-charts-toolbar__group">
            <p className="ops-charts-toolbar__label">Layout</p>
            <SegmentedControl
              items={[
                { value: 'grid', label: 'Grid (2 col)' },
                { value: 'stack', label: 'Stack (1 col)' },
              ]}
              selected={chartsLayout}
              onChange={(value) => onLayoutChange(value === 'stack' ? 'stack' : 'grid')}
            />
          </div>
        </div>
      </div>
      <div className="ops-charts-status" data-ops-status="">
        <span className="ops-charts-status__item">
          <span
            className={`ops-charts-status__dot${live ? ' ops-charts-status__dot--live' : ''}`}
            aria-hidden="true"
          />
          {live ? 'Live stream + polling' : 'Polling'}
        </span>
        <span className="ops-charts-status__item">
          Last update: <strong>{formatClockTime(lastUpdatedAt)}</strong>
        </span>
        <span className="ops-charts-status__item ops-charts-status__item--muted">
          {`Next refresh in ${formatRefreshCountdown(nextRefreshAt - now)}`}
        </span>
      </div>
      <div className={`ops-charts-grid ops-charts-grid--${chartsLayout}`} data-ops-charts-grid="">
        {specs.map((spec) => (
          <MetricChartCard key={spec.id} spec={spec} rangeHours={chartsRangeHours} />
        ))}
      </div>
    </section>
  );
}
