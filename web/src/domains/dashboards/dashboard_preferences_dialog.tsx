import { useEffect, useMemo, useState, type ReactNode, type WheelEvent } from 'react';

import { MultiSelectField } from '@/components/system/multi_select_field';
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
    <section className="ui-preferences-section ui-surface-raised grid gap-5 p-5">
      <div className="grid gap-2 border-b border-border/40 pb-4">
        <h3 className="text-sm font-semibold tracking-tight text-foreground">{title}</h3>
        {description ? <p className="text-xs leading-relaxed text-muted-foreground">{description}</p> : null}
      </div>
      <div className="grid gap-5">{children}</div>
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
        <div className="ui-preferences-dialog flex max-h-[min(80vh,44rem)] flex-col overflow-hidden">
          <DialogHeader className="ui-preferences-header shrink-0 border-b px-6 py-4 text-left">
            <DialogTitle className="text-base font-semibold tracking-tight">Preferences</DialogTitle>
          </DialogHeader>
          <div
            className="ui-preferences-dialog-body ui-scrollbar grid min-h-0 flex-1 gap-7 overflow-y-auto overscroll-y-contain px-6 py-6"
            onWheel={stopDialogWheelPropagation}
          >
            <PreferencesSection
              title="Metrics"
              description="KPI tiles and chart lines shown on the dashboard."
            >
              <MultiSelectField<DashboardMetricId>
                id="dashboard-prefs-kpi-metrics"
                label="KPI tiles"
                labelTone="nested"
                options={kpiOptions}
                value={draft.kpiMetrics}
                onChange={(value) => updateDraft('kpiMetrics', value)}
              />
              <MultiSelectField<DashboardMetricId>
                id="dashboard-prefs-chart-metrics"
                label="Chart lines"
                labelTone="nested"
                options={chartOptions}
                value={draft.chartMetrics}
                onChange={(value) => updateDraft('chartMetrics', value)}
              />
            </PreferencesSection>
            <PreferencesSection
              title="Top blocks"
              description="Breakdown tables and the columns each table shows."
            >
              <MultiSelectField
                id="dashboard-prefs-breakdown-entities"
                label="Entities"
                labelTone="nested"
                options={entityOptions}
                value={draft.breakdownEntities}
                onChange={(value) => updateDraft('breakdownEntities', value)}
              />
              <MultiSelectField
                id="dashboard-prefs-breakdown-columns"
                label="Columns"
                labelTone="nested"
                options={breakdownColumnOptions}
                value={draft.breakdownColumns}
                onChange={(value) => updateDraft('breakdownColumns', value)}
                minSelected={2}
              />
            </PreferencesSection>
            <PreferencesSection title="Recent clicks" description="Columns in the live click feed.">
              <MultiSelectField
                id="dashboard-prefs-recent-clicks"
                label="Columns"
                labelTone="nested"
                options={recentClickOptions}
                value={draft.recentClickColumns}
                onChange={(value) => updateDraft('recentClickColumns', value)}
              />
            </PreferencesSection>
          </div>
          <DialogFooter className="ui-preferences-footer shrink-0 flex-row items-center justify-between border-t px-6 py-4 sm:justify-between">
            <Button
              type="button"
              variant="link"
              className="h-auto px-0 text-muted-foreground hover:text-foreground"
              onClick={() => setDraft(defaultBuyerDashboardPreferences())}
            >
              Restore to default
            </Button>
            <div className="flex gap-3">
              <Button type="button" variant="outline" shape="square" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button
                type="button"
                shape="square"
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
