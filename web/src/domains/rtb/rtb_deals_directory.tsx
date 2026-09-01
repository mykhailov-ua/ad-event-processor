import { useState } from 'react';
import { Link } from 'react-router-dom';

import { PrimaryActionButton } from '@/components/system/action_buttons';
import { PageChrome } from '@/components/system/page_chrome';
import { EmptyState } from '@/components/system/empty_state';
import { PageSkeleton } from '@/components/system/page_skeleton';
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
import type { RtbDeal } from '@/api/types';
import { RtbNav, RtbLicenseStub, rtbPanelError } from '@/domains/rtb/rtb_nav';
import { displayMicro, displayTimestamp } from '@/lib/display';

export type RtbDealsDirectoryProps = {
  items: RtbDeal[];
  fetching: boolean;
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
    >
      <RtbNav />

      <Dialog onOpenChange={setCreateOpen} open={createOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Create deal</DialogTitle>
          </DialogHeader>
          <div className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] gap-4">
            <div className="grid gap-2">
              <Label htmlFor="rtb-create-deal-id">Deal ID</Label>
              <Input
                id="rtb-create-deal-id"
                value={draftDealId}
                onChange={(event) => onDraftDealIdChange(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="rtb-create-customer-id">Customer ID</Label>
              <Input
                id="rtb-create-customer-id"
                value={draftCustomerId}
                onChange={(event) => onDraftCustomerIdChange(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="rtb-create-floor-micro">Floor (micro)</Label>
              <Input
                id="rtb-create-floor-micro"
                value={draftFloorMicro}
                onChange={(event) => onDraftFloorMicroChange(event.target.value)}
              />
            </div>
          </div>
          {createError ? rtbPanelError(createError, 'Create failed') : null}
          <DialogFooter>
            <PrimaryActionButton loading={creating} onClick={onCreateDeal} type="button">
              Create
            </PrimaryActionButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {items.length === 0 ? (
        <EmptyState title="No deals" description="RTB deal catalog returned no entries." />
      ) : (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Deal ID</TableHead>
                <TableHead>Internal ID</TableHead>
                <TableHead>Floor (micro)</TableHead>
                <TableHead>Pacing</TableHead>
                <TableHead>Updated</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((row) => (
                <TableRow key={row.id ?? row.deal_id}>
                  <TableCell>
                    {row.id != null ? (
                      <Link className="hover:underline" to={`/rtb/deals/${row.id}`}>
                        {row.deal_id ?? ''}
                      </Link>
                    ) : (
                      row.deal_id ?? ''
                    )}
                  </TableCell>
                  <TableCell>{row.id ?? ''}</TableCell>
                  <TableCell>{displayMicro(row.floor_micro)}</TableCell>
                  <TableCell>{row.pacing ?? ''}</TableCell>
                  <TableCell>{displayTimestamp(row.updated_at)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {error && hasSnapshot ? rtbPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
