import { Link } from 'react-router-dom';
import { useEffect, useState } from 'react';

import { SUPPLY_PREVIEW_SELLERS_JSON_PATH } from '@/api/supply_api';
import { PrimaryActionButton } from '@/shell/action_buttons';
import { PageChrome } from '@/shell/page_chrome';
import { RowActionsMenu } from '@/shell/row_actions_menu';
import { EmptyState } from '@/shell/empty_state';
import { PageSkeleton } from '@/shell/page_skeleton';
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
import type { Seller } from '@/api/types';
import { CreativeNav, creativePanelError } from '@/domains/creative/creative_nav';

export type SupplySellersDirectoryProps = {
  items: Seller[];
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  draftSellerId: string;
  draftDomain: string;
  draftSellerType: string;
  draftName: string;
  editRows: Record<number, { seller_id: string; domain: string; seller_type: string; name: string }>;
  acting: boolean;
  actionError: Error | undefined;
  createSuccess: boolean;
  onDraftSellerIdChange: (value: string) => void;
  onDraftDomainChange: (value: string) => void;
  onDraftSellerTypeChange: (value: string) => void;
  onDraftNameChange: (value: string) => void;
  onEditRowChange: (
    id: number,
    field: 'seller_id' | 'domain' | 'seller_type' | 'name',
    value: string,
  ) => void;
  onCreateSeller: () => void;
  onUpdateSeller: (id: number) => void;
  onDeleteSeller: (id: number) => void;
};

export function SupplySellersDirectory({
  items,
  fetching,
  error,
  hasSnapshot,
  draftSellerId,
  draftDomain,
  draftSellerType,
  draftName,
  editRows,
  acting,
  actionError,
  createSuccess,
  onDraftSellerIdChange,
  onDraftDomainChange,
  onDraftSellerTypeChange,
  onDraftNameChange,
  onEditRowChange,
  onCreateSeller,
  onUpdateSeller,
  onDeleteSeller,
}: SupplySellersDirectoryProps) {
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
      <PageChrome title="Supply sellers">
        <CreativeNav />
        {creativePanelError(error, 'Could not load sellers')}
      </PageChrome>
    );
  }

  return (
    <PageChrome
      title="Supply sellers"
      actions={
        <PrimaryActionButton onClick={() => setCreateOpen(true)} type="button">
          Create seller
        </PrimaryActionButton>
      }
    >
      <CreativeNav />
      <Link className="text-sm text-muted-foreground hover:underline" to="/supply">
        Back to supply hub
      </Link>
      <p className="text-sm">
        <a
          className="underline"
          href={SUPPLY_PREVIEW_SELLERS_JSON_PATH}
          target="_blank"
          rel="noreferrer"
        >
          Open server sellers.json preview
        </a>
      </p>

      <Dialog onOpenChange={setCreateOpen} open={createOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Create seller</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="seller-id">Seller ID</Label>
              <Input
                id="seller-id"
                value={draftSellerId}
                onChange={(event) => onDraftSellerIdChange(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="seller-domain">Domain</Label>
              <Input
                id="seller-domain"
                value={draftDomain}
                onChange={(event) => onDraftDomainChange(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="seller-type">Type</Label>
              <Input
                id="seller-type"
                value={draftSellerType}
                onChange={(event) => onDraftSellerTypeChange(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="seller-name">Name</Label>
              <Input
                id="seller-name"
                value={draftName}
                onChange={(event) => onDraftNameChange(event.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <PrimaryActionButton loading={acting} onClick={onCreateSeller} type="button">
              Create
            </PrimaryActionButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {items.length === 0 ? (
        <EmptyState title="No sellers" description="Supply sellers table is empty." />
      ) : (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Seller ID</TableHead>
                <TableHead>Domain</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Name</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((row) => {
                const edit = editRows[row.id] ?? {
                  seller_id: row.seller_id,
                  domain: row.domain,
                  seller_type: row.seller_type,
                  name: row.name,
                };
                return (
                  <TableRow key={row.id}>
                    <TableCell>
                      <Input
                        className="font-mono text-xs"
                        value={edit.seller_id}
                        onChange={(event) =>
                          onEditRowChange(row.id, 'seller_id', event.target.value)
                        }
                      />
                    </TableCell>
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
                        value={edit.seller_type}
                        onChange={(event) =>
                          onEditRowChange(row.id, 'seller_type', event.target.value)
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <Input
                        value={edit.name}
                        onChange={(event) =>
                          onEditRowChange(row.id, 'name', event.target.value)
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <RowActionsMenu ariaLabel="Seller actions" disabled={acting}>
                        <DropdownMenuItem disabled={acting} onClick={() => onUpdateSeller(row.id)}>
                          Save
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          className="text-destructive focus:text-destructive"
                          disabled={acting}
                          onClick={() => onDeleteSeller(row.id)}
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

      {actionError ? creativePanelError(actionError, 'Seller action failed') : null}
      {error && hasSnapshot ? creativePanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
