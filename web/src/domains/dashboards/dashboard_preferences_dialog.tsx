import { useEffect, useMemo, useState, type WheelEvent } from 'react';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import type { DashboardMetricId } from '@/domains/dashboards/dashboard_metrics';
import {
  ALL_BREAKDOWN_COLUMNS,
  ALL_BREAKDOWN_ENTITIES,
  ALL_CHART_METRIC_IDS,
  ALL_KPI_METRIC_IDS,
  ALL_RECENT_CLICK_COLUMNS,
  BREAKDOWN_COLUMN_LABELS,
  BREAKDOWN_ENTITY_LABELS,
  type BuyerDashboardPreferences,
  defaultBuyerDashboardPreferences,
  KPI_METRIC_LABELS,
  RECENT_CLICK_COLUMN_LABELS,
} from '@/domains/dashboards/dashboard_preferences';
import {
  DashboardPrefsSelectionPanel,
  type DashboardPrefsOption,
} from '@/domains/dashboards/dashboard_prefs_selection_panel';

export type DashboardPreferencesDialogProps = {
  open: boolean;
  preferences: BuyerDashboardPreferences;
  onOpenChange: (open: boolean) => void;
  onApply: (preferences: BuyerDashboardPreferences) => void;
};

function toOptions<T extends string>(ids: readonly T[], labels: Record<T, string>): DashboardPrefsOption<T>[] {
  return ids.map((id) => ({ id, label: labels[id] }));
}

function stopDialogWheelPropagation(event: WheelEvent<HTMLDivElement>) {
  event.stopPropagation();
}

export function DashboardPreferencesDialog({
  open,
  preferences,
  onOpenChange,
  onApply,
}: DashboardPreferencesDialogProps) {
  const [draft, setDraft] = useState<BuyerDashboardPreferences>(preferences);

  useEffect(() => {
    if (open) {
      setDraft(preferences);
    }
  }, [open, preferences]);

  const kpiOptions = useMemo(
    () => toOptions(ALL_KPI_METRIC_IDS, KPI_METRIC_LABELS),
    [],
  );
  const chartOptions = useMemo(
    () => toOptions(ALL_CHART_METRIC_IDS, KPI_METRIC_LABELS),
    [],
  );
  const entityOptions = useMemo(
    () => toOptions(ALL_BREAKDOWN_ENTITIES, BREAKDOWN_ENTITY_LABELS),
    [],
  );
  const breakdownColumnOptions = useMemo(
    () => toOptions(ALL_BREAKDOWN_COLUMNS, BREAKDOWN_COLUMN_LABELS),
    [],
  );
  const recentClickOptions = useMemo(
    () => toOptions(ALL_RECENT_CLICK_COLUMNS, RECENT_CLICK_COLUMN_LABELS),
    [],
  );

  function updateDraft<K extends keyof BuyerDashboardPreferences>(
    key: K,
    value: BuyerDashboardPreferences[K],
  ) {
    setDraft((current) => ({ ...current, [key]: value }));
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="dashboard-preferences-dialog max-w-2xl p-0">
        <DialogHeader className="dashboard-preferences-dialog__header">
          <DialogTitle className="dashboard-preferences-dialog__title">Preferences</DialogTitle>
        </DialogHeader>

        <div
          className="dashboard-preferences-dialog__scroll ui-scrollbar"
          onWheel={stopDialogWheelPropagation}
        >
          <section className="dashboard-preferences-dialog__section">
            <h3 className="dashboard-preferences-dialog__section-title">Metrics</h3>
            <div className="dashboard-preferences-dialog__section-body">
              <DashboardPrefsSelectionPanel<DashboardMetricId>
                id="dashboard-prefs-kpi-metrics"
                label="KPI tiles"
                options={kpiOptions}
                value={draft.kpiMetrics}
                onChange={(value) => updateDraft('kpiMetrics', value)}
              />
              <DashboardPrefsSelectionPanel<DashboardMetricId>
                id="dashboard-prefs-chart-metrics"
                label="Chart lines"
                listMaxHeightClassName="max-h-40"
                options={chartOptions}
                value={draft.chartMetrics}
                onChange={(value) => updateDraft('chartMetrics', value)}
              />
              <DashboardPrefsSelectionPanel
                id="dashboard-prefs-breakdown-columns"
                label="Columns"
                listMaxHeightClassName="max-h-44"
                minSelected={2}
                options={breakdownColumnOptions}
                showChips={false}
                value={draft.breakdownColumns}
                onChange={(value) => updateDraft('breakdownColumns', value)}
              />
            </div>
          </section>

          <section className="dashboard-preferences-dialog__section">
            <h3 className="dashboard-preferences-dialog__section-title">Recent clicks</h3>
            <div className="dashboard-preferences-dialog__section-body">
              <DashboardPrefsSelectionPanel
                id="dashboard-prefs-recent-clicks"
                label="Columns"
                listMaxHeightClassName="max-h-44"
                options={recentClickOptions}
                showChips={false}
                value={draft.recentClickColumns}
                onChange={(value) => updateDraft('recentClickColumns', value)}
              />
              <DashboardPrefsSelectionPanel
                id="dashboard-prefs-breakdown-entities"
                label="Entities"
                options={entityOptions}
                value={draft.breakdownEntities}
                onChange={(value) => updateDraft('breakdownEntities', value)}
              />
            </div>
          </section>
        </div>

        <DialogFooter className="dashboard-preferences-dialog__footer">
          <button
            className="dashboard-preferences-dialog__restore"
            type="button"
            onClick={() => setDraft(defaultBuyerDashboardPreferences())}
          >
            Restore to default
          </button>
          <div className="dashboard-preferences-dialog__footer-actions">
            <Button type="button" variant="secondary" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button
              type="button"
              onClick={() => {
                onApply(draft);
                onOpenChange(false);
              }}
            >
              Apply
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
