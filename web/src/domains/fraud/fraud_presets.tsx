import { Link } from 'react-router-dom';

import { PageChrome } from '@/shell/page_chrome';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { ApiError } from '@/api/client';
import type { FraudPolicyPreset } from '@/api/types';
import { displayTimestamp } from '@/lib/display';

export type FraudPresetEditDraft = {
  pass: string;
  suspect: string;
  ivt: string;
  block: string;
};

export type FraudPresetsProps = {
  items: FraudPolicyPreset[];
  presetDrafts: Record<string, FraudPresetEditDraft>;
  fetching: boolean;
  error: Error | undefined;
  saveError: Error | undefined;
  saveSuccess: boolean;
  hasSnapshot: boolean;
  savingPresetName?: string;
  onPresetDraftChange: (name: string, patch: Partial<FraudPresetEditDraft>) => void;
  onSavePreset: (name: string) => void;
};

function presetDraftFromRow(row: FraudPolicyPreset): FraudPresetEditDraft {
  return {
    pass: row.pass != null ? String(row.pass) : '',
    suspect: row.suspect != null ? String(row.suspect) : '',
    ivt: row.ivt != null ? String(row.ivt) : '',
    block: row.block != null ? String(row.block) : '',
  };
}

function saveErrorTitle(error: Error): string {
  if (error instanceof ApiError && error.status === 403) {
    return 'Permission denied';
  }
  return 'Save failed';
}

function saveErrorMessage(error: Error): string {
  if (error instanceof ApiError && error.status === 403) {
    return 'Updating fraud presets requires the shards:write permission.';
  }
  return error.message;
}

export function FraudPresets({
  items,
  presetDrafts,
  fetching,
  error,
  saveError,
  saveSuccess,
  hasSnapshot,
  savingPresetName,
  onPresetDraftChange,
  onSavePreset,
}: FraudPresetsProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load fraud presets" message={error.message} />;
  }

  return (
    <PageChrome title="Fraud presets">
      <Link className="text-sm text-muted-foreground hover:underline" to="/fraud">
        Back to fraud hub
      </Link>

      {items.length === 0 ? (
        <EmptyState title="No presets" description="No global fraud policy presets returned." />
      ) : (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Pass</TableHead>
                <TableHead>Suspect</TableHead>
                <TableHead>IVT</TableHead>
                <TableHead>Block</TableHead>
                <TableHead>Updated</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((row) => {
                const presetName = row.name ?? '';
                const draft = presetDrafts[presetName] ?? presetDraftFromRow(row);
                const saving = savingPresetName === presetName;
                return (
                  <TableRow key={presetName || 'unknown'}>
                    <TableCell>{presetName}</TableCell>
                    <TableCell>
                      <Input
                        aria-label={`Pass threshold for ${presetName}`}
                        className="min-w-[4rem] font-mono text-xs"
                        inputMode="numeric"
                        value={draft.pass}
                        onChange={(event) =>
                          onPresetDraftChange(presetName, { pass: event.target.value })
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <Input
                        aria-label={`Suspect threshold for ${presetName}`}
                        className="min-w-[4rem] font-mono text-xs"
                        inputMode="numeric"
                        value={draft.suspect}
                        onChange={(event) =>
                          onPresetDraftChange(presetName, { suspect: event.target.value })
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <Input
                        aria-label={`IVT threshold for ${presetName}`}
                        className="min-w-[4rem] font-mono text-xs"
                        inputMode="numeric"
                        value={draft.ivt}
                        onChange={(event) =>
                          onPresetDraftChange(presetName, { ivt: event.target.value })
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <Input
                        aria-label={`Block threshold for ${presetName}`}
                        className="min-w-[4rem] font-mono text-xs"
                        inputMode="numeric"
                        value={draft.block}
                        onChange={(event) =>
                          onPresetDraftChange(presetName, { block: event.target.value })
                        }
                      />
                    </TableCell>
                    <TableCell>
                      {displayTimestamp(row.updated_at, row.updated_at_display)}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        disabled={!presetName || saving}
                        onClick={() => onSavePreset(presetName)}
                       
                        type="button"
                        variant="outline"
                      >
                        {saving ? 'Saving...' : 'Save'}
                      </Button>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}

      {saveError ? (
        <ErrorBlock title={saveErrorTitle(saveError)} message={saveErrorMessage(saveError)} />
      ) : null}
      {saveSuccess ? (
        <p className="text-sm text-muted-foreground">Preset saved. List refreshed.</p>
      ) : null}
      {error && hasSnapshot ? <ErrorBlock title="Refresh failed" message={error.message} /> : null}
    </PageChrome>
  );
}
