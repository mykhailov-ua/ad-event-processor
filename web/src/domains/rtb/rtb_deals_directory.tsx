import { useState } from 'react';
import { Link } from 'react-router-dom';

import { PrimaryActionButton } from '@/shell/action_buttons';
import { DirectoryFilterForm, FilterField } from '@/shell/filter_panel';
import { PageChrome } from '@/shell/page_chrome';
import { EmptyState } from '@/shell/empty_state';
import { PageSkeleton } from '@/shell/page_skeleton';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
  directoryTableRevalidatingClass,
} from '@/shell/directory_table';
import type { RtbDeal } from '@/api/types';
import { RtbNav, RtbLicenseStub, rtbPanelError } from '@/domains/rtb/rtb_nav';
import { displayMicro, displayTimestamp } from '@/lib/display';

export type RtbDealsDirectoryProps = {
  items?: RtbDeal[];
  fetching: boolean;
  listRevalidating?: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  licenseGated: boolean;
  draftDealId: string;
  draftCustomerId: string;
  draftFloorMicro: string;
  creating: boolean;
  createError: Error | undefined;
  onDraftDealIdChange: (value: string) => void;
  onDraftCustomerIdChange: (value: string) => void;
  onDraftFloorMicroChange: (value: string) => void;
  onCreateDeal: () => void;
};

export function RtbDealsDirectory({
  items,
  fetching,
  listRevalidating = false,
  error,
  hasSnapshot,
  licenseGated,
  draftDealId,
  draftCustomerId,
  draftFloorMicro,
  creating,
  createError,
  onDraftDealIdChange,
  onDraftCustomerIdChange,
  onDraftFloorMicroChange,
  onCreateDeal,
}: RtbDealsDirectoryProps) {
  const [createOpen, setCreateOpen] = useState(false);

  if (licenseGated) {
    return (
      <PageChrome title="RTB deals">
        <RtbNav />
        <RtbLicenseStub />
      </PageChrome>
    );
  }

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="RTB deals">
        <RtbNav />
        {rtbPanelError(error, 'Could not load RTB deals')}
      </PageChrome>
    );
  }

  return (
    <PageChrome
      title="RTB deals"
      actions={
        <PrimaryActionButton onClick={() => setCreateOpen(true)} type="button">
          Create deal
        </PrimaryActionButton>
      }
      controlPanel={<RtbNav />}
    >
      <Dialog onOpenChange={setCreateOpen} open={createOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Create deal</DialogTitle>
          </DialogHeader>
          <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
            <FilterField htmlFor="rtb-create-deal-id" label="Deal ID">
              <Input
                id="rtb-create-deal-id"
                value={draftDealId}
                onChange={(event) => onDraftDealIdChange(event.target.value)}
              />
            </FilterField>
            <FilterField htmlFor="rtb-create-customer-id" label="Customer ID">
              <Input
                id="rtb-create-customer-id"
                value={draftCustomerId}
                onChange={(event) => onDraftCustomerIdChange(event.target.value)}
              />
            </FilterField>
            <FilterField htmlFor="rtb-create-floor-micro" label="Floor (micro)">
              <Input
                id="rtb-create-floor-micro"
                value={draftFloorMicro}
                onChange={(event) => onDraftFloorMicroChange(event.target.value)}
              />
            </FilterField>
          </DirectoryFilterForm>
          {createError ? rtbPanelError(createError, 'Create failed') : null}
          <DialogFooter>
            <PrimaryActionButton loading={creating} onClick={onCreateDeal} type="button">
              Create
            </PrimaryActionButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {(items ?? []).length === 0 ? (
        <EmptyState title="No deals" description="RTB deal catalog returned no entries." />
      ) : (
        <DirectoryTable className={directoryTableRevalidatingClass(listRevalidating)}>
          <TableHeader>
            <TableRow>
              <DirectoryTableHead>Deal ID</DirectoryTableHead>
              <DirectoryTableHead>Internal ID</DirectoryTableHead>
              <DirectoryTableHead>Floor (micro)</DirectoryTableHead>
              <DirectoryTableHead>Pacing</DirectoryTableHead>
              <DirectoryTableHead>Updated</DirectoryTableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(items ?? []).map((row) => (
              <TableRow key={row.id ?? row.deal_id}>
                <TableCell>
                  {row.id != null ? (
                    <Link className="hover:underline" to={`/rtb/deals/${row.id}`}>
                      {row.deal_id ?? ''}
                    </Link>
                  ) : (
                    (row.deal_id ?? '')
                  )}
                </TableCell>
                <TableCell>{row.id ?? ''}</TableCell>
                <TableCell>{displayMicro(row.floor_micro)}</TableCell>
                <TableCell>{row.pacing ?? ''}</TableCell>
                <TableCell>{displayTimestamp(row.updated_at)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </DirectoryTable>
      )}

      {error && hasSnapshot ? rtbPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
