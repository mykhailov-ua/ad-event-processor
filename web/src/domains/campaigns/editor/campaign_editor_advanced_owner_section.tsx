import { FilterField, INLINE_FILTER_ACTION_GRID_CLASS } from '@/shell/filter_panel';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { editorApiErrorBlock } from '@/domains/campaigns/editor/campaign_editor_shared';
import { adminTypography } from '@/lib/admin_kit';

type CampaignEditorAdvancedOwnerSectionProps = {
  draftOwnerUserId: string;
  onDraftOwnerUserIdChange: (value: string) => void;
  transferringOwner: boolean;
  fetching: boolean;
  ownerError: Error | undefined;
  ownerSuccess: boolean;
  onTransferOwner: () => void;
  exporting: boolean;
  exportError: Error | undefined;
  onExportCampaign: () => void;
};

export function CampaignEditorAdvancedOwnerSection({
  draftOwnerUserId,
  onDraftOwnerUserIdChange,
  transferringOwner,
  fetching,
  ownerError,
  ownerSuccess,
  onTransferOwner,
  exporting,
  exportError,
  onExportCampaign,
}: CampaignEditorAdvancedOwnerSectionProps) {
  return (
    <section className="flex flex-col gap-4" >
      <h2 className={adminTypography.sectionTitle}>Owner and export</h2>
      <div className={INLINE_FILTER_ACTION_GRID_CLASS} >
        <FilterField htmlFor="campaign-owner-user-id" label="New owner user ID">
          <Input
            id="campaign-owner-user-id"
            value={draftOwnerUserId}
            disabled={transferringOwner || fetching}
            onChange={(event) => onDraftOwnerUserIdChange(event.target.value)}
          />
        </FilterField>
        <Button type="button" disabled={transferringOwner || fetching} onClick={onTransferOwner}>
          {transferringOwner ? 'Transferring...' : 'Transfer owner'}
        </Button>
      </div>
      {ownerSuccess ? (
        <p  className={adminTypography.bodyMuted} role="status">
          Owner transfer accepted.
        </p>
      ) : null}
      {ownerError
        ? editorApiErrorBlock(ownerError, 'Owner transfer unavailable', 'Could not transfer owner')
        : null}

      <div className="flex justify-start gap-2" >
        <Button
          type="button"
          variant="secondary"
          disabled={exporting || fetching}
          onClick={onExportCampaign}
        >
          {exporting ? 'Exporting...' : 'Download export bundle'}
        </Button>
      </div>
      {exportError
        ? editorApiErrorBlock(exportError, 'Export unavailable', 'Could not export campaign')
        : null}
    </section>
  );
}
