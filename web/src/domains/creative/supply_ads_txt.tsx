import { Link } from 'react-router-dom';
import { useEffect, useState } from 'react';

import { SUPPLY_PREVIEW_ADS_TXT_PATH } from '@/api/supply_api';
import { PrimaryActionButton } from '@/components/system/action_buttons';
import { PageChrome } from '@/components/system/page_chrome';
import { RowActionsMenu } from '@/components/system/row_actions_menu';
import { EmptyState } from '@/components/system/empty_state';
import { PageSkeleton } from '@/components/system/page_skeleton';
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
import type { AdsTxtEntry } from '@/api/types';
import { CreativeNav, creativePanelError } from '@/domains/creative/creative_nav';

export type SupplyAdsTxtDirectoryProps = {
  items: AdsTxtEntry[];
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  draftDomain: string;
  draftAccountId: string;
  draftRelationship: string;
  draftSortOrder: string;
  editRows: Record<
    number,
    { domain: string; publisher_account_id: string; relationship: string; sort_order: string }
  >;
  acting: boolean;
  actionError: Error | undefined;
  createSuccess: boolean;
  onDraftDomainChange: (value: string) => void;
  onDraftAccountIdChange: (value: string) => void;
  onDraftRelationshipChange: (value: string) => void;
  onDraftSortOrderChange: (value: string) => void;
  onEditRowChange: (
    id: number,
    field: 'domain' | 'publisher_account_id' | 'relationship' | 'sort_order',
    value: string,
  ) => void;
  onCreateRow: () => void;
  onUpdateRow: (id: number) => void;
  onDeleteRow: (id: number) => void;
};

export function SupplyAdsTxtDirectory({
  items,
  fetching,
  error,
  hasSnapshot,
  draftDomain,
  draftAccountId,
  draftRelationship,
  draftSortOrder,
  editRows,
  acting,
  actionError,
  createSuccess,
  onDraftDomainChange,
  onDraftAccountIdChange,
  onDraftRelationshipChange,
  onDraftSortOrderChange,
  onEditRowChange,
  onCreateRow,
  onUpdateRow,
  onDeleteRow,
}: SupplyAdsTxtDirectoryProps) {
  const [createOpen, setCreateOpen] = useState(false);

  useEffect(() => {
    if (createSuccess) {
      setCreateOpen(false);
    }
  }, [createSuccess]);

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Supply ads.txt">
        <CreativeNav />
        {creativePanelError(error, 'Could not load ads.txt rows')}
      </PageChrome>
    );
  }

  return (
    <PageChrome
      title="Supply ads.txt"
      actions={
        <PrimaryActionButton onClick={() => setCreateOpen(true)} type="button">
          Create ads.txt row
        </PrimaryActionButton>
      }
    >
      <CreativeNav />
      <Link className="text-sm text-muted-foreground hover:underline" to="/supply">
        Back to supply hub
      </Link>
      <p className="text-sm">
        <a className="underline" href={SUPPLY_PREVIEW_ADS_TXT_PATH} target="_blank" rel="noreferrer">
          Open server ads.txt preview
        </a>
      </p>

      <Dialog onOpenChange={setCreateOpen} open={createOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Create ads.txt row</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="ads-domain">Domain</Label>
              <Input
                id="ads-domain"
                value={draftDomain}
                onChange={(event) => onDraftDomainChange(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="ads-account">Publisher account ID</Label>
              <Input
                id="ads-account"
                value={draftAccountId}
                onChange={(event) => onDraftAccountIdChange(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="ads-relationship">Relationship</Label>
              <Input
                id="ads-relationship"
                value={draftRelationship}
                onChange={(event) => onDraftRelationshipChange(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="ads-sort-order">Sort order</Label>
              <Input
                id="ads-sort-order"
                value={draftSortOrder}
                onChange={(event) => onDraftSortOrderChange(event.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <PrimaryActionButton loading={acting} onClick={onCreateRow} type="button">
              Create
            </PrimaryActionButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {items.length === 0 ? (
        <EmptyState title="No ads.txt rows" description="Supply ads.txt table is empty." />
      ) : (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Domain</TableHead>
                <TableHead>Account</TableHead>
                <TableHead>Relationship</TableHead>
                <TableHead>Order</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((row) => {
                const edit = editRows[row.id] ?? {
                  domain: row.domain,
                  publisher_account_id: row.publisher_account_id,
                  relationship: row.relationship,
                  sort_order: String(row.sort_order ?? ''),
                };
                return (
                  <TableRow key={row.id}>
                    <TableCell>
                      <Input
                        value={edit.domain}
                        onChange={(event) =>
                          onEditRowChange(row.id, 'domain', event.target.value)
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <Input
                        className="font-mono text-xs"
                        value={edit.publisher_account_id}
                        onChange={(event) =>
                          onEditRowChange(row.id, 'publisher_account_id', event.target.value)
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <Input
                        value={edit.relationship}
                        onChange={(event) =>
                          onEditRowChange(row.id, 'relationship', event.target.value)
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <Input
                        value={edit.sort_order}
                        onChange={(event) =>
                          onEditRowChange(row.id, 'sort_order', event.target.value)
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <RowActionsMenu ariaLabel="Row actions" disabled={acting}>
                        <DropdownMenuItem disabled={acting} onClick={() => onUpdateRow(row.id)}>
                          Save
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          className="text-destructive focus:text-destructive"
                          disabled={acting}
                          onClick={() => onDeleteRow(row.id)}
                        >
                          Delete
                        </DropdownMenuItem>
                      </RowActionsMenu>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}

      {actionError ? creativePanelError(actionError, 'ads.txt action failed') : null}
      {error && hasSnapshot ? creativePanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
