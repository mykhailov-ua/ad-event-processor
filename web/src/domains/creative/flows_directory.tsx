import { useState } from 'react';
import { useRunWhenTrue } from '@/hooks/use_run_when_true';
import { Link } from 'react-router-dom';

import { PrimaryActionButton } from '@/shell/action_buttons';
import { PageChrome } from '@/shell/page_chrome';
import { EmptyState } from '@/shell/empty_state';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { FilterField } from '@/shell/filter_panel';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import type { Flow, Lander, Offer } from '@/api/types';
import { CreativeDirectoryStack } from '@/domains/creative/creative_directory_stack';
import { creativePanelError } from '@/domains/creative/creative_nav';
import { FlowEditorVisual } from '@/domains/creative/flow_editor_visual';
import {
  validateVisualPathWeights,
  type FlowPathVisualRow,
} from '@/domains/creative/flow_path_model';
import { ErrorBlock } from '@/shell/error_block';
import { TableHost } from '@/shell/ui_bands';
import { displayTimestamp } from '@/lib/display';

export type FlowsDirectoryProps = {
  items?: Flow[];
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  landers: Lander[];
  offers: Offer[];
  draftName?: string;
  draftRows?: FlowPathVisualRow[];
  creating?: boolean;
  createError?: Error;
  createSuccess?: boolean;
  onDraftNameChange?: (value: string) => void;
  onDraftRowsChange?: (rows: FlowPathVisualRow[]) => void;
  onCreateFlow?: () => void;
};

export function FlowsDirectory({
  items,
  fetching,
  error,
  hasSnapshot,
  landers,
  offers,
  draftName = '',
  draftRows = [],
  creating = false,
  createError,
  createSuccess = false,
  onDraftNameChange,
  onDraftRowsChange,
  onCreateFlow,
}: FlowsDirectoryProps) {
  const [createOpen, setCreateOpen] = useState(false);
  const validationError = validateVisualPathWeights(draftRows);

  useRunWhenTrue(createSuccess, () => setCreateOpen(false));

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={3} />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Flows">
        <CreativeDirectoryStack>
          {creativePanelError(error, 'Could not load flows')}
        </CreativeDirectoryStack>
      </PageChrome>
    );
  }

  return (
    <PageChrome
      title="Flows"
      actions={
        onCreateFlow ? (
          <Button onClick={() => setCreateOpen(true)} type="button">
            Create flow
          </Button>
        ) : undefined
      }
    >
      <CreativeDirectoryStack>
        {onCreateFlow ? (
          <Dialog onOpenChange={setCreateOpen} open={createOpen}>
            <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
              <DialogHeader>
                <DialogTitle>Create flow</DialogTitle>
              </DialogHeader>
              <div className="grid gap-4">
                <FilterField htmlFor="flow-create-name" label="Name">
                  <Input
                    id="flow-create-name"
                    placeholder="Flow name..."
                    value={draftName}
                    onChange={(event) => onDraftNameChange?.(event.target.value)}
                  />
                </FilterField>
                <FlowEditorVisual
                  disabled={creating}
                  landers={landers}
                  offers={offers}
                  rows={draftRows}
                  validationError={validationError ?? undefined}
                  onRowsChange={(rows) => onDraftRowsChange?.(rows)}
                />
                {createError ? (
                  <ErrorBlock message={createError.message} title="Could not create flow" />
                ) : null}
              </div>
              <DialogFooter>
                <PrimaryActionButton loading={creating} onClick={onCreateFlow} type="button">
                  Create flow
                </PrimaryActionButton>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        ) : null}

        <div aria-atomic="true" aria-live="polite">
          {(items ?? []).length === 0 ? (
            <EmptyState
              variant="blank-slate"
              title="No flows"
              description="Create a flow to split traffic across landers and offers."
              actionLabel={onCreateFlow ? 'Create flow' : undefined}
              onAction={onCreateFlow ? () => setCreateOpen(true) : undefined}
            />
          ) : (
            <TableHost>
              <DirectoryTable nested>
                <TableHeader>
                  <TableRow>
                    <DirectoryTableHead>Name</DirectoryTableHead>
                    <DirectoryTableHead>Paths</DirectoryTableHead>
                    <DirectoryTableHead>Created</DirectoryTableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(items ?? []).map((row) => (
                    <TableRow key={row.id}>
                      <TableCell>
                        <Link className="hover:underline" to={`/flows/${row.id}`}>
                          {row.name}
                        </Link>
                      </TableCell>
                      <TableCell>{row.paths?.length ?? 0}</TableCell>
                      <TableCell>{displayTimestamp(row.created_at)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </DirectoryTable>
            </TableHost>
          )}
        </div>

        {error && hasSnapshot ? creativePanelError(error, 'Refresh failed') : null}
      </CreativeDirectoryStack>
    </PageChrome>
  );
}
