import { X } from 'lucide-react';

import { Checkbox } from '@/components/ui/checkbox';
import {
  dashboardPrefsCheckboxLabelClass,
  dashboardPrefsCheckboxListClass,
  dashboardPrefsCheckboxRowClass,
  dashboardPrefsChipBoxClass,
  dashboardPrefsChipClass,
  dashboardPrefsChipEmptyClass,
  dashboardPrefsChipLabelClass,
  dashboardPrefsChipRemoveClass,
  dashboardPrefsColumnSummaryClass,
  dashboardPrefsFieldClass,
  dashboardPrefsFieldLabelClass,
} from '@/domains/dashboards/dashboard_preferences_classes';
import { cn } from '@/lib/utils';

export type DashboardPrefsOption<T extends string> = {
  id: T;
  label: string;
};

export type DashboardPrefsSelectionPanelProps<T extends string> = {
  id: string;
  label: string;
  options: readonly DashboardPrefsOption<T>[];
  value: readonly T[];
  onChange: (value: T[]) => void;
  minSelected?: number;
  listMaxHeightClassName?: string;
  showChips?: boolean;
  summary?: string;
};

function toggleSelection<T extends string>(
  current: readonly T[],
  optionId: T,
  checked: boolean,
  minSelected: number,
): T[] {
  if (checked) {
    if (current.includes(optionId)) {
      return [...current];
    }
    return [...current, optionId];
  }
  if (current.length <= minSelected) {
    return [...current];
  }
  return current.filter((item) => item !== optionId);
}

export function formatDashboardPrefsSummary<T extends string>(
  value: readonly T[],
  labels: Record<T, string>,
): string {
  return value.map((id) => labels[id]).join(', ');
}

export function DashboardPrefsSelectionPanel<T extends string>({
  id,
  label,
  options,
  value,
  onChange,
  minSelected = 1,
  listMaxHeightClassName = 'max-h-52',
  showChips = true,
  summary,
}: DashboardPrefsSelectionPanelProps<T>) {
  const labelById = new Map(options.map((option) => [option.id, option.label]));
  const labels = Object.fromEntries(options.map((option) => [option.id, option.label])) as Record<T, string>;
  const selectedSet = new Set(value);
  const resolvedSummary = summary ?? formatDashboardPrefsSummary(value, labels);

  function removeChip(optionId: T) {
    if (value.length <= minSelected) {
      return;
    }
    onChange(value.filter((item) => item !== optionId));
  }

  return (
    <div className={dashboardPrefsFieldClass}>
      <p className={dashboardPrefsFieldLabelClass} id={`${id}-label`}>
        {label}
      </p>
      {showChips ? (
        <div
          aria-labelledby={`${id}-label`}
          className={dashboardPrefsChipBoxClass}
          role="group"
        >
          {value.length === 0 ? (
            <span className={dashboardPrefsChipEmptyClass}>No items selected</span>
          ) : (
            value.map((optionId) => {
              const canRemove = value.length > minSelected;
              return (
                <span key={optionId} className={dashboardPrefsChipClass}>
                  <span className={dashboardPrefsChipLabelClass}>
                    {labelById.get(optionId) ?? optionId}
                  </span>
                  {canRemove ? (
                    <button
                      aria-label={`Remove ${labelById.get(optionId) ?? optionId}`}
                      className={dashboardPrefsChipRemoveClass}
                      type="button"
                      onClick={() => removeChip(optionId)}
                    >
                      <X aria-hidden className="h-3 w-3" />
                    </button>
                  ) : null}
                </span>
              );
            })
          )}
        </div>
      ) : (
        <p
          aria-live="polite"
          className={dashboardPrefsColumnSummaryClass}
          title={resolvedSummary}
        >
          {resolvedSummary}
        </p>
      )}
      <div
        aria-labelledby={`${id}-label`}
        className={cn(dashboardPrefsCheckboxListClass, listMaxHeightClassName)}
        role="group"
      >
        {options.map((option) => {
          const checked = selectedSet.has(option.id);
          return (
            <label key={option.id} className={dashboardPrefsCheckboxRowClass}>
              <Checkbox
                checked={checked}
                id={`${id}-${option.id}`}
                onCheckedChange={(nextChecked) => {
                  onChange(toggleSelection(value, option.id, nextChecked, minSelected));
                }}
              />
              <span className={dashboardPrefsCheckboxLabelClass}>{option.label}</span>
            </label>
          );
        })}
      </div>
    </div>
  );
}
