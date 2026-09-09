import { useCallback, useEffect, useMemo, useState } from 'react';

import { FilterApplyButton, SecondaryActionButton } from '@/shell/action_buttons';
import { DirectoryListMeta } from '@/shell/directory_list_meta';
import { DirectoryFilterForm, FilterField, FilterPanel } from '@/shell/filter_panel';
import { DirectoryPageShell, DirectoryMutationError } from '@/shell/directory_page_shell';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { AuditLog } from '@/api/types';
import type { AuditAuthSourceFilter } from '@/domains/audit/use_audit_page_workspace';
import { AuditSelectionPanel } from '@/domains/audit/audit_selection_panel';
import { ControlPlaneSelectTable } from '@/shell/control_plane_select_table';

export type AuditDirectoryProps = {
  items?: AuditLog[];
  total: number;
  limit: number;
  offset: number;
  fetching: boolean;
  listRevalidating?: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  draftAdminId: string;
  draftTargetId: string;
  draftAction: string;
  draftAuthSource: AuditAuthSourceFilter;
  draftApiKeyId: string;
  draftRedactPii: boolean;
  exporting: boolean;
  exportError: Error | undefined;
  exportTruncated: boolean;
  exportNextCursor?: string;
  onDraftAdminIdChange: (value: string) => void;
  onDraftTargetIdChange: (value: string) => void;
  onDraftActionChange: (value: string) => void;
  onDraftAuthSourceChange: (value: AuditAuthSourceFilter) => void;
  onDraftApiKeyIdChange: (value: string) => void;
  onDraftRedactPiiChange: (value: boolean) => void;
  onApplyFilters: () => void;
  onExportCsv: () => void;
  onPageChange: (nextOffset: number) => void;
};

function auditRowId(row: AuditLog, index: number): string {
  return row.id ?? `${row.created_at ?? 'row'}-${row.action ?? 'action'}-${index}`;
}

function auditRowLabel(row: AuditLog): string {
  const action = row.action ?? 'action';
  const target = row.target_type ?? 'target';
  return `${action} · ${target}`;
}

export function AuditDirectory({
  items,
  total,
  limit,
  offset,
  fetching,
  listRevalidating = false,
  error,
  hasSnapshot,
  draftAdminId,
  draftTargetId,
  draftAction,
  draftAuthSource,
  draftApiKeyId,
  draftRedactPii,
  exporting,
  exportError,
  exportTruncated,
  exportNextCursor,
  onDraftAdminIdChange,
  onDraftTargetIdChange,
  onDraftActionChange,
  onDraftAuthSourceChange,
  onDraftApiKeyIdChange,
  onDraftRedactPiiChange,
  onApplyFilters,
  onExportCsv,
  onPageChange,
}: AuditDirectoryProps) {
  const [selectedEntryId, setSelectedEntryId] = useState<string | null>(null);
  const canGoPrev = offset > 0;
  const canGoNext = offset + limit < total;

  const operateRows = useMemo(
    () =>
      (items ?? []).map((row, index) => ({
        id: auditRowId(row, index),
        label: auditRowLabel(row),
      })),
    [items]
  );

  const selectedEntry = useMemo(() => {
    if (!selectedEntryId) {
      return undefined;
    }
    return (items ?? []).find((row, index) => auditRowId(row, index) === selectedEntryId);
  }, [items, selectedEntryId]);

  useEffect(() => {
    setSelectedEntryId(null);
  }, [offset, limit, draftAdminId, draftTargetId, draftAction, draftAuthSource, draftApiKeyId]);

  useEffect(() => {
    if (selectedEntryId && !selectedEntry) {
      setSelectedEntryId(null);
    }
  }, [selectedEntry, selectedEntryId]);

  const handleClearSelection = useCallback(() => {
    setSelectedEntryId(null);
  }, []);

  return (
    <DirectoryPageShell
      alerts={
        <>
          <DirectoryMutationError error={exportError} title="Export failed" />
          {exportTruncated ? (
            <p role="status">
              Export truncated.{exportNextCursor ? ` Next cursor: ${exportNextCursor}` : ''}
            </p>
          ) : null}
        </>
      }
      aside={
        <AuditSelectionPanel selectedEntry={selectedEntry} onClearSelection={handleClearSelection} />
      }
      blockingErrorTitle="Could not load audit log"
      controlPanel={
        <FilterPanel>
          <DirectoryFilterForm
            layout="auto-fill"
            onSubmit={(event) => {
              event.preventDefault();
              onApplyFilters();
            }}
          >
            <FilterField htmlFor="audit-filter-admin-id" label="Admin ID">
              <Input
                id="audit-filter-admin-id"
                value={draftAdminId}
                onChange={(event) => onDraftAdminIdChange(event.target.value)}
              />
            </FilterField>
            <FilterField htmlFor="audit-filter-target-id" label="Target ID">
              <Input
                id="audit-filter-target-id"
                value={draftTargetId}
                onChange={(event) => onDraftTargetIdChange(event.target.value)}
              />
            </FilterField>
            <FilterField htmlFor="audit-filter-action" label="Action">
              <Input
                id="audit-filter-action"
                value={draftAction}
                onChange={(event) => onDraftActionChange(event.target.value)}
              />
            </FilterField>
            <FilterField htmlFor="audit-filter-auth-source" label="Auth source">
              <Select
                value={draftAuthSource || 'any'}
                onValueChange={(value) =>
                  onDraftAuthSourceChange(value === 'any' ? '' : (value as AuditAuthSourceFilter))
                }
              >
                <SelectTrigger id="audit-filter-auth-source">
                  <SelectValue placeholder="Any" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="any">Any</SelectItem>
                  <SelectItem value="session">Session</SelectItem>
                  <SelectItem value="api_key">API key</SelectItem>
                </SelectContent>
              </Select>
            </FilterField>
            <FilterField htmlFor="audit-filter-api-key-id" label="API key ID">
              <Input
                id="audit-filter-api-key-id"
                value={draftApiKeyId}
                onChange={(event) => onDraftApiKeyIdChange(event.target.value)}
              />
            </FilterField>
            <FilterApplyButton disabled={fetching}>Apply</FilterApplyButton>
            <div>
              <Checkbox
                checked={draftRedactPii}
                id="audit-redact-pii"
                onCheckedChange={(checked) => onDraftRedactPiiChange(checked === true)}
              />
              <Label htmlFor="audit-redact-pii">Redact PII in export</Label>
            </div>
            <SecondaryActionButton
              disabled={exporting}
              loading={exporting}
              onClick={onExportCsv}
              type="button"
            >
              Export CSV
            </SecondaryActionButton>
          </DirectoryFilterForm>
          <DirectoryListMeta>
            {total > 0
              ? `Showing ${offset + 1}-${Math.min(offset + (items ?? []).length, total)} of ${total}`
              : 'No audit entries'}
          </DirectoryListMeta>
        </FilterPanel>
      }
      fetchState={{ fetching, error, hasSnapshot }}
      footer={
        <DirectoryPaginationFooter
          canGoNext={canGoNext}
          canGoPrev={canGoPrev}
          disabled={fetching}
          onNext={() => onPageChange(offset + limit)}
          onPrev={() => onPageChange(Math.max(0, offset - limit))}
        />
      }
      skeletonColumns={2}
      title="Audit"
    >
      <ControlPlaneSelectTable
        disabled={fetching}
        emptyMessage="Admin actions will appear here when recorded."
        nameColumnLabel="Entry"
        revalidating={listRevalidating}
        rows={operateRows}
        selectedId={selectedEntryId}
        onSelectedIdChange={setSelectedEntryId}
      />
    </DirectoryPageShell>
  );
}
