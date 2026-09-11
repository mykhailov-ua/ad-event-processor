import { useMemo } from 'react';

import type { SavedView } from '@/api/types';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { DashboardPanelSection } from '@/domains/dashboards/dashboard_panel_section';
import { adminTypography } from '@/lib/admin_kit';
import { adminSpacing } from '@/lib/admin_spacing';
import { ErrorBlock } from '@/shell/error_block';
import { FilterField, FilterPanel } from '@/shell/filter_panel';
import { BentoSection } from '@/shell/bento_card';
import {
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';

export type ExportHubSavedViewsProps = {
  views: SavedView[];
  viewsError?: Error;
  viewsFetching: boolean;
  canManage: boolean;
  canExport: boolean;
  presetName: string;
  selectedViewId: string;
  saving: boolean;
  exportingViewId?: string;
  onPresetNameChange: (value: string) => void;
  onSelectedViewIdChange: (value: string) => void;
  onLoadView: () => void;
  onSaveView: () => void;
  onDeleteView: () => void;
  onExportView: () => void;
};

export function ExportHubSavedViews({
  views,
  viewsError,
  viewsFetching,
  canManage,
  canExport,
  presetName,
  selectedViewId,
  saving,
  exportingViewId,
  onPresetNameChange,
  onSelectedViewIdChange,
  onLoadView,
  onSaveView,
  onDeleteView,
  onExportView,
}: ExportHubSavedViewsProps) {
  const selectedView = useMemo(
    () => views.find((view) => view.id === selectedViewId),
    [selectedViewId, views]
  );

  return (
    <BentoSection title="Saved presets">
      {viewsError ? <ErrorBlock error={viewsError} title="Saved presets load failed" /> : null}
      {!canManage ? (
        <p className={adminTypography.bodyMuted}>
          Read-only: save and delete require campaigns:write. Export shortcut requires exports:run.
        </p>
      ) : null}

      <FilterPanel aria-label="Saved export preset">
        <FilterField htmlFor="export-hub-preset-name" label="Preset name">
          <Input
            disabled={!canManage || saving}
            id="export-hub-preset-name"
            placeholder="Weekly placements CSV"
            value={presetName}
            onChange={(event) => onPresetNameChange(event.target.value)}
          />
        </FilterField>
        <FilterField htmlFor="export-hub-preset-select" label="Saved preset">
          <Select
            disabled={viewsFetching || views.length === 0}
            value={selectedViewId || undefined}
            onValueChange={onSelectedViewIdChange}
          >
            <SelectTrigger id="export-hub-preset-select">
              <SelectValue placeholder={viewsFetching ? 'Loading presets...' : 'Select preset'} />
            </SelectTrigger>
            <SelectContent>
              {views.map((view) => (
                <SelectItem key={view.id} value={view.id ?? ''}>
                  {view.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </FilterField>
        <div className={adminSpacing.flex.buttonGroup}>
          <Button disabled={!selectedViewId} type="button" variant="outline" onClick={onLoadView}>
            Load
          </Button>
          <Button disabled={!canManage || saving} type="button" onClick={onSaveView}>
            {saving ? 'Saving...' : selectedView ? 'Update' : 'Save'}
          </Button>
          <Button
            disabled={!canManage || !selectedViewId || saving}
            type="button"
            variant="destructive"
            onClick={onDeleteView}
          >
            Delete
          </Button>
          <Button
            disabled={
              !canExport ||
              !selectedViewId ||
              Boolean(exportingViewId) ||
              selectedView?.report_key === 'billing-export' ||
              selectedView?.report_key === 'audit-export'
            }
            type="button"
            onClick={onExportView}
          >
            {exportingViewId ? 'Exporting...' : 'Export now'}
          </Button>
        </div>
      </FilterPanel>

      {views.length > 0 ? (
        <DashboardPanelSection tableAriaLabel="Saved export presets">
          <TableHeader>
            <TableRow>
              <DirectoryTableHead>Name</DirectoryTableHead>
              <DirectoryTableHead>Report key</DirectoryTableHead>
              <DirectoryTableHead>Updated</DirectoryTableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {views.map((view) => (
              <TableRow key={view.id}>
                <TableCell>{view.name}</TableCell>
                <TableCell>{view.report_key}</TableCell>
                <TableCell>{view.updated_at ?? ''}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </DashboardPanelSection>
      ) : null}
    </BentoSection>
  );
}
