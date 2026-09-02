import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';

import { PrimaryActionButton } from '@/shell/action_buttons';
import { PageChrome } from '@/shell/page_chrome';
import { EmptyState } from '@/shell/empty_state';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Badge } from '@/components/ui/badge';
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { Lander } from '@/api/types';
import { CreativeNav, creativePanelError } from '@/domains/creative/creative_nav';
import { displayTimestamp } from '@/lib/display';

export type LandersDirectoryProps = {
  items: Lander[];
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  draftName?: string;
  draftUrl?: string;
  creating?: boolean;
  createError?: Error;
  createSuccess?: boolean;
  onDraftNameChange?: (value: string) => void;
  onDraftUrlChange?: (value: string) => void;
  onCreateLander?: () => void;
};

export function LandersDirectory({
  items,
  fetching,
  error,
  hasSnapshot,
  draftName = '',
  draftUrl = '',
  creating = false,
  createError,
  createSuccess = false,
  onDraftNameChange,
  onDraftUrlChange,
  onCreateLander,
}: LandersDirectoryProps) {
  const [createOpen, setCreateOpen] = useState(false);

  useEffect(() => {
    if (createSuccess) {
      setCreateOpen(false);
    }
  }, [createSuccess]);

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={4} />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Landers">
        <CreativeNav />
        {creativePanelError(error, 'Could not load landers')}
      </PageChrome>
    );
  }

  return (
    <PageChrome
      title="Landers"
      actions={
        onCreateLander ? (
          <Button className="text-sm" onClick={() => setCreateOpen(true)} type="button">
            Create lander
          </Button>
        ) : undefined
      }
    >
      <CreativeNav />

      {onCreateLander ? (
        <Dialog onOpenChange={setCreateOpen} open={createOpen}>
          <DialogContent className="max-w-lg">
            <DialogHeader>
              <DialogTitle>Create lander</DialogTitle>
            </DialogHeader>
            <div className="grid gap-4">
              <div className="grid gap-2">
                <Label htmlFor="lander-create-name">Name</Label>
                <Input
                  id="lander-create-name"
                  placeholder="Lander name..."
                  value={draftName}
                  onChange={(event) => onDraftNameChange?.(event.target.value)}
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="lander-create-url">URL</Label>
                <Input
                  id="lander-create-url"
                  placeholder="https://..."
                  value={draftUrl}
                  onChange={(event) => onDraftUrlChange?.(event.target.value)}
                />
              </div>
              {createError ? creativePanelError(createError, 'Could not create lander') : null}
            </div>
            <DialogFooter>
              <PrimaryActionButton loading={creating} onClick={onCreateLander} type="button">
                Create lander
              </PrimaryActionButton>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      ) : null}

      <div aria-atomic="true" aria-live="polite">
        {items.length === 0 ? (
          <EmptyState
            variant="blank-slate"
            title="No landers"
            description="Create a lander page to route traffic from flows."
            actionLabel={onCreateLander ? 'Create lander' : undefined}
            onAction={onCreateLander ? () => setCreateOpen(true) : undefined}
          />
        ) : (
          <div className="ui-table-frame">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>URL</TableHead>
                  <TableHead>Hosted</TableHead>
                  <TableHead>Created</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell>
                      {row.hosted_asset_id ? (
                        <Link className="hover:underline" to={`/landers/${row.id}/editor`}>
                          {row.name}
                        </Link>
                      ) : (
                        row.name
                      )}
                    </TableCell>
                    <TableCell className="max-w-md truncate">{row.url ?? row.hosted_url ?? ''}</TableCell>
                    <TableCell>
                      {row.hosted_asset_id ? <Badge variant="outline">hosted</Badge> : ''}
                    </TableCell>
                    <TableCell className="tabular-nums">{displayTimestamp(row.created_at)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>

      {error && hasSnapshot ? creativePanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
