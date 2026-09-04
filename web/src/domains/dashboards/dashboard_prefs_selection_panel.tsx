import { X } from 'lucide-react';

import { Checkbox } from '@/components/ui/checkbox';
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
    <div className="dashboard-preferences-dialog__field">
      <p className="dashboard-preferences-dialog__field-label" id={`${id}-label`}>
        {label}
      </p>
      {showChips ? (
        <div
          aria-labelledby={`${id}-label`}
          className="dashboard-preferences-dialog__chip-box"
          role="group"
        >
          {value.length === 0 ? (
            <span className="dashboard-preferences-dialog__chip-empty">No items selected</span>
          ) : (
            value.map((optionId) => {
              const canRemove = value.length > minSelected;
              return (
                <span key={optionId} className="dashboard-preferences-dialog__chip">
                  <span className="dashboard-preferences-dialog__chip-label">
                    {labelById.get(optionId) ?? optionId}
                  </span>
                  {canRemove ? (
                    <button
                      aria-label={`Remove ${labelById.get(optionId) ?? optionId}`}
                      className="dashboard-preferences-dialog__chip-remove"
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
          className="dashboard-preferences-dialog__column-summary"
          title={resolvedSummary}
        >
          {resolvedSummary}
        </p>
      )}
      <div
        aria-labelledby={`${id}-label`}
        className={cn('dashboard-preferences-dialog__checkbox-list ui-scrollbar', listMaxHeightClassName)}
        role="group"
      >
        {options.map((option) => {
          const checked = selectedSet.has(option.id);
          return (
            <label key={option.id} className="dashboard-preferences-dialog__checkbox-row">
              <Checkbox
                checked={checked}
                id={`${id}-${option.id}`}
                onCheckedChange={(nextChecked) => {
                  onChange(toggleSelection(value, option.id, nextChecked, minSelected));
                }}
              />
              <span className="dashboard-preferences-dialog__checkbox-label">{option.label}</span>
            </label>
          );
        })}
      </div>
    </div>
  );
}
