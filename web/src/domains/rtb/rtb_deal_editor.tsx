import { Link } from 'react-router-dom';

import type { RtbDealUpdateSpec } from '@/api/types';
import { PageChrome } from '@/components/system/page_chrome';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import type { RtbDeal } from '@/api/types';
import { RtbNav, RtbLicenseStub, rtbPanelError } from '@/domains/rtb/rtb_nav';

export type RtbDealEditorProps = {
  deal: RtbDeal | undefined;
  draft: RtbDealUpdateSpec;
  fetching: boolean;
  saving: boolean;
  deleting: boolean;
  error: Error | undefined;
  saveError: Error | undefined;
  deleteError: Error | undefined;
  hasSnapshot: boolean;
  licenseGated: boolean;
  onDraftChange: (patch: Partial<RtbDealUpdateSpec>) => void;
  onSave: () => void;
  onDelete: () => void;
};

export function RtbDealEditor({
  deal,
  draft,
  fetching,
  saving,
  deleting,
  error,
  saveError,
  deleteError,
  hasSnapshot,
  licenseGated,
  onDraftChange,
  onSave,
  onDelete,
}: RtbDealEditorProps) {
  if (licenseGated) {
    return (
      <PageChrome title="RTB deal">
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
      <PageChrome title="RTB deal">
        <RtbNav />
        {rtbPanelError(error, 'Could not load RTB deal')}
      </PageChrome>
    );
  }

  if (!deal) {
    return (
      <PageChrome title="RTB deal">
        <RtbNav />
        {rtbPanelError(new Error('Deal not found'), 'Could not load RTB deal')}
      </PageChrome>
    );
  }

  return (
    <PageChrome title={`Deal ${deal.deal_id ?? deal.id ?? ''}`}>
      <RtbNav />
      <Link className="text-sm text-muted-foreground hover:underline" to="/rtb/deals">
        Back to deals
      </Link>

      <form
        className="grid max-w-xl gap-4"
        onSubmit={(event) => {
          event.preventDefault();
          onSave();
        }}
      >
        <div className="grid gap-2">
          <Label htmlFor="rtb-deal-id">deal_id</Label>
          <Input
            id="rtb-deal-id"
            value={draft.deal_id ?? ''}
            onChange={(event) => onDraftChange({ deal_id: event.target.value })}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="rtb-floor-micro">floor_micro</Label>
          <Input
            id="rtb-floor-micro"
            value={draft.floor_micro != null ? String(draft.floor_micro) : ''}
            onChange={(event) =>
              onDraftChange({
                floor_micro: event.target.value ? Number(event.target.value) : undefined,
              })
            }
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="rtb-geo-mask">geo_mask</Label>
          <Input
            id="rtb-geo-mask"
            value={draft.geo_mask != null ? String(draft.geo_mask) : ''}
            onChange={(event) =>
              onDraftChange({
                geo_mask: event.target.value ? Number(event.target.value) : undefined,
              })
            }
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="rtb-cat-mask">cat_mask</Label>
          <Input
            id="rtb-cat-mask"
            value={draft.cat_mask != null ? String(draft.cat_mask) : ''}
            onChange={(event) =>
              onDraftChange({
                cat_mask: event.target.value ? Number(event.target.value) : undefined,
              })
            }
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="rtb-pacing">pacing</Label>
          <Input
            id="rtb-pacing"
            value={draft.pacing ?? ''}
            onChange={(event) => onDraftChange({ pacing: event.target.value })}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="rtb-seats">seats</Label>
          <Input
            id="rtb-seats"
            value={draft.seats != null ? String(draft.seats) : ''}
            onChange={(event) =>
              onDraftChange({
                seats: event.target.value ? Number(event.target.value) : undefined,
              })
            }
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="rtb-customer-id">customer_id</Label>
          <Input
            id="rtb-customer-id"
            value={draft.customer_id ?? ''}
            onChange={(event) => onDraftChange({ customer_id: event.target.value })}
          />
        </div>

        <div className="flex flex-wrap gap-2">
          <Button disabled={saving} type="submit">
            Save PATCH
          </Button>
          <Button disabled={deleting} onClick={onDelete} type="button" variant="destructive">
            Delete
          </Button>
        </div>
      </form>

      {saveError ? rtbPanelError(saveError, 'Save failed') : null}
      {deleteError ? rtbPanelError(deleteError, 'Delete failed') : null}
      {error && hasSnapshot ? rtbPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
