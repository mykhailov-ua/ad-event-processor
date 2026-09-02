import { Button } from '@/components/ui/button';
import { useEffect, useMemo, useState, type ReactNode, type WheelEvent } from 'react';

import { MultiSelectField } from '@/shell/multi_select_field';
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

export type DashboardPreferencesDialogProps = {
  open: boolean;
  preferences: BuyerDashboardPreferences;
  onOpenChange: (open: boolean) => void;
  onApply: (preferences: BuyerDashboardPreferences) => void;
};

function toOptions<T extends string>(ids: readonly T[], labels: Record<T, string>) {
  return ids.map((id) => ({ id, label: labels[id] }));
}

function stopDialogWheelPropagation(event: WheelEvent<HTMLDivElement>) {
  event.stopPropagation();
}

function PreferencesSection({
  title,
  description,
  children,
}: {
  title: string;
  description?: string;
  children: ReactNode;
}) {
  return (
    <section className="admin-stack">
      <div className="admin-stack admin-stack--compact">
        <h3 className="admin-ops-block__title">{title}</h3>
        {description ? <p className="admin-muted">{description}</p> : null}
      </div>
      <div className="admin-stack admin-stack--compact">{children}</div>
    </section>
  );
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
      <DialogContent className="max-w-2xl p-0">
        <div
          className="admin-preferences-dialog ui-scrollbar max-h-[min(80vh,44rem)] overflow-y-auto overscroll-y-contain"
          onWheel={stopDialogWheelPropagation}
        >
          <DialogHeader className="admin-preferences-dialog__header sticky top-0 z-10 shrink-0 border-b px-4 py-3 text-left">
            <DialogTitle className="admin-ops-block__title">Preferences</DialogTitle>
          </DialogHeader>
          <div className="admin-preferences-dialog__body grid gap-4 px-4 py-4">
            <PreferencesSection
              description="KPI tiles and chart lines shown on the dashboard."
              title="Metrics"
            >
              <MultiSelectField<DashboardMetricId>
                id="dashboard-prefs-kpi-metrics"
                label="KPI tiles"
                options={kpiOptions}
                value={draft.kpiMetrics}
                onChange={(value) => updateDraft('kpiMetrics', value)}
              />
              <MultiSelectField<DashboardMetricId>
                id="dashboard-prefs-chart-metrics"
                label="Chart lines"
                options={chartOptions}
                value={draft.chartMetrics}
                onChange={(value) => updateDraft('chartMetrics', value)}
              />
            </PreferencesSection>
            <PreferencesSection
              description="Breakdown tables and the columns each table shows."
              title="Top blocks"
            >
              <MultiSelectField
                id="dashboard-prefs-breakdown-entities"
                label="Entities"
                options={entityOptions}
                value={draft.breakdownEntities}
                onChange={(value) => updateDraft('breakdownEntities', value)}
              />
              <MultiSelectField
                id="dashboard-prefs-breakdown-columns"
                label="Columns"
                minSelected={2}
                options={breakdownColumnOptions}
                value={draft.breakdownColumns}
                onChange={(value) => updateDraft('breakdownColumns', value)}
              />
            </PreferencesSection>
            <PreferencesSection description="Columns in the live click feed." title="Recent clicks">
              <MultiSelectField
                id="dashboard-prefs-recent-clicks"
                label="Columns"
                options={recentClickOptions}
                value={draft.recentClickColumns}
                onChange={(value) => updateDraft('recentClickColumns', value)}
              />
            </PreferencesSection>
          </div>
          <DialogFooter className="admin-preferences-dialog__footer sticky bottom-0 z-10 shrink-0 flex-row items-center justify-between border-t px-4 py-3 sm:justify-between">
            <button
              className="admin-text-link"
              type="button"
              onClick={() => setDraft(defaultBuyerDashboardPreferences())}
            >
              Restore to default
            </button>
            <div className="admin-toolbar-group">
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
        </div>
      </DialogContent>
    </Dialog>
  );
}
