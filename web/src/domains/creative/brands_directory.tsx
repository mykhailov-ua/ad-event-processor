import { useState } from 'react';
import { useRunWhenTrue } from '@/hooks/use_run_when_true';
import { Link } from 'react-router-dom';

import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import { PageChrome } from '@/shell/page_chrome';
import { CustomerScopeBar } from '@/shell/customer_scope_bar';
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
import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
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
import type { Brand } from '@/api/types';
import { CreativeDirectoryStack } from '@/domains/creative/creative_directory_stack';
import { creativePanelError } from '@/domains/creative/creative_nav';
import { RowActionsMenu } from '@/shell/row_actions_menu';
import { TableHost } from '@/shell/ui_bands';
import { displayTimestamp } from '@/lib/display';

export type BrandsDirectoryProps = {
  items?: Brand[];
  appliedCustomerId: string;
  draftCustomerId: string;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  onDraftCustomerIdChange: (value: string) => void;
  onApplyCustomerScope: () => void;
  draftBrandName?: string;
  creating?: boolean;
  createError?: Error;
  createSuccess?: boolean;
  onDraftBrandNameChange?: (value: string) => void;
  onCreateBrand?: () => void;
  editingBrand?: Brand | null;
  editBrandName?: string;
  saving?: boolean;
  editError?: Error;
  onOpenEditBrand?: (brand: Brand) => void;
  onCloseEditBrand?: () => void;
  onEditBrandNameChange?: (value: string) => void;
  onSaveBrand?: () => void;
  actingBrandId?: string | null;
  actionError?: Error;
  onDeleteBrand?: (brand: Brand) => void;
};

export function BrandsDirectory({
  items,
  appliedCustomerId,
  draftCustomerId,
  fetching,
  error,
  hasSnapshot,
  onDraftCustomerIdChange,
  onApplyCustomerScope,
  draftBrandName = '',
  creating = false,
  createError,
  createSuccess = false,
  onDraftBrandNameChange,
  onCreateBrand,
  editingBrand = null,
  editBrandName = '',
  saving = false,
  editError,
  onOpenEditBrand,
  onCloseEditBrand,
  onEditBrandNameChange,
  onSaveBrand,
  actingBrandId = null,
  actionError,
  onDeleteBrand,
}: BrandsDirectoryProps) {
  const [createOpen, setCreateOpen] = useState(false);
  const editOpen = editingBrand != null;
  const acting = actingBrandId != null || saving;

  useRunWhenTrue(createSuccess, () => setCreateOpen(false));

  if (!appliedCustomerId) {
    return (
      <PageChrome title="Brands">
        <CreativeDirectoryStack>
        <CustomerScopeBar
          appliedCustomerId={appliedCustomerId}
          draftCustomerId={draftCustomerId}
          onApply={onApplyCustomerScope}
          onDraftCustomerIdChange={onDraftCustomerIdChange}
        />
        <EmptyState title="Customer required" description="Apply a customer ID to list brands." />
        </CreativeDirectoryStack>
      </PageChrome>
    );
  }

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={3} />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Brands">
        <CreativeDirectoryStack>
        <CustomerScopeBar
          appliedCustomerId={appliedCustomerId}
          draftCustomerId={draftCustomerId}
          onApply={onApplyCustomerScope}
          onDraftCustomerIdChange={onDraftCustomerIdChange}
        />
        {creativePanelError(error, 'Could not load brands')}
        </CreativeDirectoryStack>
      </PageChrome>
    );
  }

  return (
    <PageChrome
      title="Brands"
      actions={
        onCreateBrand ? (
          <Button onClick={() => setCreateOpen(true)} type="button">
            Create brand
          </Button>
        ) : undefined
      }
    >
      <CreativeDirectoryStack>
      <CustomerScopeBar
        appliedCustomerId={appliedCustomerId}
        draftCustomerId={draftCustomerId}
        onApply={onApplyCustomerScope}
        onDraftCustomerIdChange={onDraftCustomerIdChange}
      />

      {onCreateBrand ? (
        <Dialog onOpenChange={setCreateOpen} open={createOpen}>
          <DialogContent className="max-w-md">
            <DialogHeader>
              <DialogTitle>Create brand</DialogTitle>
            </DialogHeader>
            <FilterField htmlFor="brand-create-name" label="Brand name">
              <Input
                id="brand-create-name"
                value={draftBrandName}
                onChange={(event) => onDraftBrandNameChange?.(event.target.value)}
              />
            </FilterField>
            {createError ? (
              <div>{creativePanelError(createError, 'Could not create brand')}</div>
            ) : null}
            <DialogFooter>
              <PrimaryActionButton loading={creating} onClick={onCreateBrand} type="button">
                Create brand
              </PrimaryActionButton>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      ) : null}

      {onSaveBrand && onCloseEditBrand ? (
        <Dialog
          onOpenChange={(open) => {
            if (!open) {
              onCloseEditBrand();
            }
          }}
          open={editOpen}
        >
          <DialogContent className="max-w-md">
            <DialogHeader>
              <DialogTitle>Edit brand</DialogTitle>
            </DialogHeader>
            <FilterField htmlFor="brand-edit-name" label="Brand name">
              <Input
                id="brand-edit-name"
                value={editBrandName}
                onChange={(event) => onEditBrandNameChange?.(event.target.value)}
              />
            </FilterField>
            {editError ? (
              <div>{creativePanelError(editError, 'Could not update brand')}</div>
            ) : null}
            <DialogFooter>
              <SecondaryActionButton onClick={onCloseEditBrand} type="button">
                Cancel
              </SecondaryActionButton>
              <PrimaryActionButton loading={saving} onClick={onSaveBrand} type="button">
                Save brand
              </PrimaryActionButton>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      ) : null}

      <div aria-atomic="true" aria-live="polite">
        {(items ?? []).length === 0 ? (
          <EmptyState
            variant="blank-slate"
            title="No brands"
            description="Create a brand to organize creatives for this customer."
            actionLabel={onCreateBrand ? 'Create brand' : undefined}
            onAction={onCreateBrand ? () => setCreateOpen(true) : undefined}
          />
        ) : (
          <TableHost>
            <DirectoryTable nested>
            <TableHeader>
              <TableRow>
                <DirectoryTableHead>Name</DirectoryTableHead>
                <DirectoryTableHead>Freq limit</DirectoryTableHead>
                <DirectoryTableHead>Updated</DirectoryTableHead>
                <DirectoryTableHead className="w-12" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {(items ?? []).map((row) => (
                <TableRow key={row.id}>
                  <TableCell>
                    <Link
                      className="hover:underline"
                      state={{ brandName: row.name }}
                      to={`/brand-creatives/${row.id}`}
                    >
                      {row.name}
                    </Link>
                  </TableCell>
                  <TableCell>
                    {row.freq_limit}/{row.freq_window}
                  </TableCell>
                  <TableCell>{displayTimestamp(row.updated_at)}</TableCell>
                  <TableCell>
                    {onOpenEditBrand || onDeleteBrand ? (
                      <RowActionsMenu
                        disabled={acting && actingBrandId === row.id}
                        ariaLabel={`Actions for ${row.name}`}
                      >
                        {onOpenEditBrand ? (
                          <DropdownMenuItem onClick={() => onOpenEditBrand(row)}>
                            Edit
                          </DropdownMenuItem>
                        ) : null}
                        {onDeleteBrand ? (
                          <DropdownMenuItem onClick={() => onDeleteBrand(row)}>
                            Delete
                          </DropdownMenuItem>
                        ) : null}
                      </RowActionsMenu>
                    ) : null}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </DirectoryTable>
          </TableHost>
        )}
      </div>

      {actionError ? creativePanelError(actionError, 'Brand action failed') : null}
      {error && hasSnapshot ? creativePanelError(error, 'Refresh failed') : null}
      </CreativeDirectoryStack>
    </PageChrome>
  );
}
