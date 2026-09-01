import { useEffect, useState } from 'react';

import { PrimaryActionButton } from '@/components/system/action_buttons';
import { PageChrome } from '@/components/system/page_chrome';
import { EmptyState } from '@/components/system/empty_state';
import { PageSkeleton } from '@/components/system/page_skeleton';
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
import type { Offer } from '@/api/types';
import { CreativeNav, creativePanelError } from '@/domains/creative/creative_nav';
import { displayTimestamp } from '@/lib/display';

export type OffersDirectoryProps = {
  items: Offer[];
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
  onCreateOffer?: () => void;
};

export function OffersDirectory({
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
  onCreateOffer,
}: OffersDirectoryProps) {
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
      <PageChrome title="Offers">
        <CreativeNav />
        {creativePanelError(error, 'Could not load offers')}
      </PageChrome>
    );
  }

  return (
    <PageChrome
      title="Offers"
      actions={
        onCreateOffer ? (
          <Button className="h-9 text-sm" onClick={() => setCreateOpen(true)} type="button">
            Create offer
          </Button>
        ) : undefined
      }
    >
      <CreativeNav />

      {onCreateOffer ? (
        <Dialog onOpenChange={setCreateOpen} open={createOpen}>
          <DialogContent className="max-w-lg">
            <DialogHeader>
              <DialogTitle>Create offer</DialogTitle>
            </DialogHeader>
            <div className="grid gap-4">
              <div className="grid gap-2">
                <Label htmlFor="offer-create-name">Name</Label>
                <Input
                  id="offer-create-name"
                  placeholder="Offer name…"
                  value={draftName}
                  onChange={(event) => onDraftNameChange?.(event.target.value)}
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="offer-create-url">URL</Label>
                <Input
                  id="offer-create-url"
                  placeholder="https://…"
                  value={draftUrl}
                  onChange={(event) => onDraftUrlChange?.(event.target.value)}
                />
              </div>
              {createError ? creativePanelError(createError, 'Could not create offer') : null}
            </div>
            <DialogFooter>
              <PrimaryActionButton loading={creating} onClick={onCreateOffer} type="button">
                Create offer
              </PrimaryActionButton>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      ) : null}

      <div aria-atomic="true" aria-live="polite">
        {items.length === 0 ? (
          <EmptyState
            variant="blank-slate"
            title="No offers"
            description="Create an offer to link landers and flows."
            actionLabel={onCreateOffer ? 'Create offer' : undefined}
            onAction={onCreateOffer ? () => setCreateOpen(true) : undefined}
          />
        ) : (
          <div className="ui-table-frame">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>URL</TableHead>
                  <TableHead>Created</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell>{row.name}</TableCell>
                    <TableCell className="max-w-md truncate">{row.url}</TableCell>
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
