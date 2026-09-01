import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';

import { PrimaryActionButton } from '@/components/system/action_buttons';
import { PageChrome } from '@/components/system/page_chrome';
import { RowActionsMenu } from '@/components/system/row_actions_menu';
import { EmptyState } from '@/components/system/empty_state';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { Badge } from '@/components/ui/badge';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
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
import type { BrandCreative } from '@/api/types';
import { CreativeNav, creativePanelError } from '@/domains/creative/creative_nav';
import { displayTimestamp } from '@/lib/display';

export type BrandCreativesDirectoryProps = {
  brandId: string;
  items: BrandCreative[];
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  draftName: string;
  draftUrl: string;
  draftWeight: string;
  draftStatus: string;
  acting: boolean;
  actionError: Error | undefined;
  actionSuccess: boolean;
  editingCreative?: BrandCreative;
  editName: string;
  editUrl: string;
  editWeight: string;
  editStatus: string;
  editSuccess: boolean;
  onDraftNameChange: (value: string) => void;
  onDraftUrlChange: (value: string) => void;
  onDraftWeightChange: (value: string) => void;
  onDraftStatusChange: (value: string) => void;
  onEditNameChange: (value: string) => void;
  onEditUrlChange: (value: string) => void;
  onEditWeightChange: (value: string) => void;
  onEditStatusChange: (value: string) => void;
  onCreateCreative: () => void;
  onOpenEditCreative: (creative: BrandCreative) => void;
  onCloseEditCreative: () => void;
  onSaveCreative: () => void;
  onDeleteCreative: (creativeId: string) => void;
};

export function BrandCreativesDirectory({
  brandId,
  items,
  fetching,
  error,
  hasSnapshot,
  draftName,
  draftUrl,
  draftWeight,
  draftStatus,
  acting,
  actionError,
  actionSuccess,
  editingCreative,
  editName,
  editUrl,
  editWeight,
  editStatus,
  editSuccess,
  onDraftNameChange,
  onDraftUrlChange,
  onDraftWeightChange,
  onDraftStatusChange,
  onEditNameChange,
  onEditUrlChange,
  onEditWeightChange,
  onEditStatusChange,
  onCreateCreative,
  onOpenEditCreative,
  onCloseEditCreative,
  onSaveCreative,
  onDeleteCreative,
}: BrandCreativesDirectoryProps) {
  const [createOpen, setCreateOpen] = useState(false);

  useEffect(() => {
    if (createSuccess) {
      setCreateOpen(false);
    }
  }, [createSuccess]);

  useEffect(() => {
    if (editSuccess) {
      onCloseEditCreative();
    }
  }, [editSuccess, onCloseEditCreative]);

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Brand creatives">
        <CreativeNav />
        {creativePanelError(error, 'Could not load brand creatives')}
      </PageChrome>
    );
  }

  return (
    <PageChrome
      title="Brand creatives"
      actions={
        <PrimaryActionButton onClick={() => setCreateOpen(true)} type="button">
          Create creative
        </PrimaryActionButton>
      }
    >
      <CreativeNav />
      <Link className="text-sm text-muted-foreground hover:underline" to="/brands">
        Back to brands
      </Link>
      <p className="text-sm text-muted-foreground">
        Brand ID: <span className="font-mono text-xs text-foreground">{brandId}</span>
      </p>

      <Dialog onOpenChange={setCreateOpen} open={createOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Create creative</DialogTitle>
          </DialogHeader>
          <div className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] gap-4">
            <div className="grid gap-2">
              <Label htmlFor="creative-name">Name</Label>
              <Input id="creative-name" value={draftName} onChange={(e) => onDraftNameChange(e.target.value)} />
            </div>
            <div className="grid gap-2 md:col-span-2">
              <Label htmlFor="creative-url">Landing URL</Label>
              <Input id="creative-url" value={draftUrl} onChange={(e) => onDraftUrlChange(e.target.value)} />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="creative-weight">Weight</Label>
              <Input id="creative-weight" value={draftWeight} onChange={(e) => onDraftWeightChange(e.target.value)} />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="creative-status">Status</Label>
              <Input id="creative-status" value={draftStatus} onChange={(e) => onDraftStatusChange(e.target.value)} />
            </div>
          </div>
          {actionError && createOpen ? creativePanelError(actionError, 'Creative action failed') : null}
          <DialogFooter>
            <PrimaryActionButton loading={acting} onClick={onCreateCreative} type="button">
              Create creative
            </PrimaryActionButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        onOpenChange={(open) => {
          if (!open) {
            onCloseEditCreative();
          }
        }}
        open={editingCreative != null}
      >
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Edit creative</DialogTitle>
          </DialogHeader>
          <div className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] gap-4">
            <div className="grid gap-2">
              <Label htmlFor="creative-edit-name">Name</Label>
              <Input
                id="creative-edit-name"
                value={editName}
                onChange={(e) => onEditNameChange(e.target.value)}
              />
            </div>
            <div className="grid gap-2 md:col-span-2">
              <Label htmlFor="creative-edit-url">Landing URL</Label>
              <Input
                id="creative-edit-url"
                value={editUrl}
                onChange={(e) => onEditUrlChange(e.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="creative-edit-weight">Weight</Label>
              <Input
                id="creative-edit-weight"
                value={editWeight}
                onChange={(e) => onEditWeightChange(e.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="creative-edit-status">Status</Label>
              <Input
                id="creative-edit-status"
                value={editStatus}
                onChange={(e) => onEditStatusChange(e.target.value)}
              />
            </div>
          </div>
          {actionError && editingCreative ? creativePanelError(actionError, 'Could not save creative') : null}
          <DialogFooter>
            <PrimaryActionButton loading={acting} onClick={onSaveCreative} type="button">
              Save creative
            </PrimaryActionButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {items.length === 0 ? (
        <EmptyState title="No creatives" description="This brand has no creatives yet." />
      ) : (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Weight</TableHead>
                <TableHead>Landing URL</TableHead>
                <TableHead>Updated</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((row) => (
                <TableRow key={row.id}>
                  <TableCell>{row.name}</TableCell>
                  <TableCell>
                    <Badge variant="outline">{row.status}</Badge>
                  </TableCell>
                  <TableCell>{row.weight}</TableCell>
                  <TableCell className="max-w-md truncate">{row.landing_url}</TableCell>
                  <TableCell>{displayTimestamp(row.updated_at)}</TableCell>
                  <TableCell>
                    <RowActionsMenu ariaLabel="Creative actions" disabled={acting}>
                      <DropdownMenuItem
                        disabled={acting}
                        onClick={() => onOpenEditCreative(row)}
                      >
                        Edit
                      </DropdownMenuItem>
                      <DropdownMenuItem
                        className="text-destructive focus:text-destructive"
                        disabled={acting}
                        onClick={() => onDeleteCreative(row.id)}
                      >
                        Delete
                      </DropdownMenuItem>
                    </RowActionsMenu>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {error && hasSnapshot ? creativePanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
