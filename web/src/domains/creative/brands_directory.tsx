import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';

import { PrimaryActionButton } from '@/components/system/action_buttons';
import { PageChrome } from '@/components/system/page_chrome';
import { CustomerScopeBar } from '@/components/system/customer_scope_bar';
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
import type { Brand } from '@/api/types';
import { CreativeNav, creativePanelError } from '@/domains/creative/creative_nav';
import { displayTimestamp } from '@/lib/display';

export type BrandsDirectoryProps = {
  items: Brand[];
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
}: BrandsDirectoryProps) {
  const [createOpen, setCreateOpen] = useState(false);

  useEffect(() => {
    if (createSuccess) {
      setCreateOpen(false);
    }
  }, [createSuccess]);

  if (!appliedCustomerId) {
    return (
      <PageChrome title="Brands">
        <CreativeNav />
        <CustomerScopeBar
          appliedCustomerId={appliedCustomerId}
          draftCustomerId={draftCustomerId}
          onApply={onApplyCustomerScope}
          onDraftCustomerIdChange={onDraftCustomerIdChange}
        />
        <EmptyState
          title="Customer required"
          description="Apply a customer ID to list brands."
        />
      </PageChrome>
    );
  }

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={3} />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Brands">
        <CreativeNav />
        <CustomerScopeBar
          appliedCustomerId={appliedCustomerId}
          draftCustomerId={draftCustomerId}
          onApply={onApplyCustomerScope}
          onDraftCustomerIdChange={onDraftCustomerIdChange}
        />
        {creativePanelError(error, 'Could not load brands')}
      </PageChrome>
    );
  }

  return (
    <PageChrome
      title="Brands"
      actions={
        onCreateBrand ? (
          <Button className="h-9 text-sm" onClick={() => setCreateOpen(true)} type="button">
            Create brand
          </Button>
        ) : undefined
      }
    >
      <CreativeNav />

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
              <div className="grid gap-2">
                <Label htmlFor="brand-create-name">Brand name</Label>
                <Input
                  id="brand-create-name"
                  value={draftBrandName}
                  onChange={(event) => onDraftBrandNameChange?.(event.target.value)}
                />
              </div>
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

      <div aria-atomic="true" aria-live="polite">
      {items.length === 0 ? (
        <EmptyState
          variant="blank-slate"
          title="No brands"
          description="Create a brand to organize creatives for this customer."
          actionLabel={onCreateBrand ? 'Create brand' : undefined}
          onAction={onCreateBrand ? () => setCreateOpen(true) : undefined}
        />
      ) : (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Freq limit</TableHead>
                <TableHead>Updated</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((row) => (
                <TableRow key={row.id}>
                  <TableCell>
                    <Link className="hover:underline" to={`/brand-creatives/${row.id}`}>
                      {row.name}
                    </Link>
                  </TableCell>
                  <TableCell>
                    {row.freq_limit}/{row.freq_window}
                  </TableCell>
                  <TableCell>{displayTimestamp(row.updated_at)}</TableCell>
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
