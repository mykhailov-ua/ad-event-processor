import { useCallback, useEffect, useMemo, useState } from 'react';
import { to } from '../lib/to.js';
import { can } from '../helpers/permissions.js';
import * as auth from '../helpers/auth.js';
import {
  createSavedView,
  deleteSavedView,
  listSavedViews,
  type SavedViewRow,
} from '../helpers/report_api.js';
import { Button } from './button.js';
import { FormField } from './form_field.js';

export type ReportSavedViewsPanelProps = {
  reportKey: string;
  customerId: string;
  currentSpec: Record<string, unknown>;
  onApply: (spec: Record<string, unknown>) => void;
};

/**
 * Saved report presets for a single report page: load, save, and delete views.
 */
export function ReportSavedViewsPanel({
  reportKey,
  customerId,
  currentSpec,
  onApply,
}: ReportSavedViewsPanelProps) {
  const user = auth.getUser();
  const canWrite = can(user?.permissions ?? [], 'campaigns:write');
  const [views, setViews] = useState<SavedViewRow[]>([]);
  const [selectedId, setSelectedId] = useState('');
  const [saveName, setSaveName] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const filteredViews = useMemo(
    () => views.filter((row) => (row.report_key ?? '') === reportKey),
    [views, reportKey]
  );

  const reload = useCallback(async () => {
    if (!customerId) {
      setViews([]);
      return;
    }
    setLoading(true);
    setError(null);
    const [rows, err] = await to(listSavedViews(customerId));
    setLoading(false);
    if (err) {
      setError(err instanceof Error ? err.message : 'Failed to load saved views');
      return;
    }
    setViews((rows ?? []) as SavedViewRow[]);
  }, [customerId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  const parseSpec = (raw: SavedViewRow['spec']): Record<string, unknown> => {
    if (!raw) return {};
    if (typeof raw === 'string') {
      try {
        return JSON.parse(raw) as Record<string, unknown>;
      } catch {
        return {};
      }
    }
    return raw;
  };

  /** Applies the selected saved view filters to the report form. */
  const handleLoad = () => {
    const row = filteredViews.find((v) => v.id === selectedId);
    if (!row) return;
    onApply(parseSpec(row.spec));
  };

  /** Persists the current filter state as a named saved view. */
  const handleSave = async () => {
    const name = saveName.trim();
    if (!name || !customerId) return;
    setLoading(true);
    setError(null);
    const [, err] = await to(
      createSavedView({
        customerId,
        name,
        reportKey,
        spec: currentSpec,
      })
    );
    setLoading(false);
    if (err) {
      setError(err instanceof Error ? err.message : 'Failed to save view');
      return;
    }
    setSaveName('');
    await reload();
  };

  /** Removes the selected saved view. */
  const handleDelete = async () => {
    if (!selectedId) return;
    setLoading(true);
    setError(null);
    const [, err] = await to(deleteSavedView(selectedId));
    setLoading(false);
    if (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete view');
      return;
    }
    setSelectedId('');
    await reload();
  };

  if (!customerId) return null;

  return (
    <div className="report-saved-views mb-3" data-testid="report-saved-views">
      <div className="form-row items-end">
        <FormField label="Saved views" htmlFor="report-saved-view-select">
          <select
            id="report-saved-view-select"
            className="form-input"
            value={selectedId}
            disabled={loading || filteredViews.length === 0}
            onChange={(e) => setSelectedId(e.target.value)}
          >
            <option value="">Select preset...</option>
            {filteredViews.map((view) => (
              <option key={view.id ?? view.name} value={view.id ?? ''}>
                {view.name ?? view.id}
              </option>
            ))}
          </select>
        </FormField>
        <Button
          label="Load"
          variant="secondary"
          size="sm"
          disabled={!selectedId || loading}
          onClick={handleLoad}
        />
        {canWrite ? (
          <>
            <FormField label="Save as" htmlFor="report-saved-view-name">
              <input
                id="report-saved-view-name"
                className="form-input form-input--sm"
                value={saveName}
                placeholder="Preset name"
                disabled={loading}
                onChange={(e) => setSaveName(e.target.value)}
              />
            </FormField>
            <Button
              label="Save view"
              variant="secondary"
              size="sm"
              icon="bookmark"
              disabled={loading || !saveName.trim()}
              data-testid="report-save-view-button"
              onClick={() => void handleSave()}
            />
            <Button
              label="Delete"
              variant="danger"
              size="sm"
              disabled={!selectedId || loading}
              onClick={() => void handleDelete()}
            />
          </>
        ) : null}
      </div>
      {error ? <p className="text-danger text-sm mt-1">{error}</p> : null}
    </div>
  );
}
