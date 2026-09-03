import { useEffect, useState } from 'react';
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
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import type { Flow } from '@/api/types';
import { CreativeNav, creativePanelError } from '@/domains/creative/creative_nav';
import { displayTimestamp } from '@/lib/display';

export type FlowsDirectoryProps = {
  items: Flow[];
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  draftName?: string;
  draftPathsJson?: string;
  creating?: boolean;
  createError?: Error;
  createSuccess?: boolean;
  onDraftNameChange?: (value: string) => void;
  onDraftPathsJsonChange?: (value: string) => void;
  onCreateFlow?: () => void;
};

export function FlowsDirectory({
  items,
  fetching,
  error,
  hasSnapshot,
  draftName = '',
  draftPathsJson = '[{"weight":100,"lander_id":"","offer_id":""}]',
  creating = false,
  createError,
  createSuccess = false,
  onDraftNameChange,
  onDraftPathsJsonChange,
  onCreateFlow,
}: FlowsDirectoryProps) {
  const [createOpen, setCreateOpen] = useState(false);

  useEffect(() => {
    if (createSuccess) {
      setCreateOpen(false);
    }
  }, [createSuccess]);

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={3} />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Flows">
        <CreativeNav />
        {creativePanelError(error, 'Could not load flows')}
      </PageChrome>
    );
  }

  return (
    <PageChrome
      title="Flows"
      actions={
        onCreateFlow ? (
          <Button className="text-sm" onClick={() => setCreateOpen(true)} type="button">
            Create flow
          </Button>
        ) : undefined
      }
    >
      <CreativeNav />

      {onCreateFlow ? (
        <Dialog onOpenChange={setCreateOpen} open={createOpen}>
          <DialogContent className="max-w-lg">
            <DialogHeader>
              <DialogTitle>Create flow</DialogTitle>
            </DialogHeader>
            <div className="grid gap-4">
              <div className="grid gap-2">
                <Label htmlFor="flow-create-name">Name</Label>
                <Input
                  id="flow-create-name"
                  placeholder="Flow name..."
                  value={draftName}
                  onChange={(event) => onDraftNameChange?.(event.target.value)}
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="flow-create-paths">Paths JSON</Label>
                <Textarea
                  id="flow-create-paths"
                  className="min-h-24 font-mono text-sm"
                  placeholder='[{"weight":100,"lander_id":"","offer_id":""}]'
                  value={draftPathsJson}
                  onChange={(event) => onDraftPathsJsonChange?.(event.target.value)}
                />
              </div>
              {createError ? creativePanelError(createError, 'Could not create flow') : null}
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
        {items.length === 0 ? (
          <EmptyState
            variant="blank-slate"
            title="No flows"
            description="Create a flow to split traffic across landers and offers."
            actionLabel={onCreateFlow ? 'Create flow' : undefined}
            onAction={onCreateFlow ? () => setCreateOpen(true) : undefined}
          />
        ) : (
          <DirectoryTable>
              <TableHeader>
                <TableRow>
                  <DirectoryTableHead>Name</DirectoryTableHead>
                  <DirectoryTableHead>Paths</DirectoryTableHead>
                  <DirectoryTableHead>Created</DirectoryTableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell>
                      <Link className="hover:underline" to={`/flows/${row.id}`}>
                        {row.name}
                      </Link>
                    </TableCell>
                    <TableCell className="tabular-nums">{row.paths?.length ?? 0}</TableCell>
                    <TableCell className="tabular-nums">{displayTimestamp(row.created_at)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </DirectoryTable>
        )}
      </div>

      {error && hasSnapshot ? creativePanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
