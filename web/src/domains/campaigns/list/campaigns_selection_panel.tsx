import { Link } from 'react-router-dom';

import type { Campaign } from '@/api/types';
import { Button } from '@/components/ui/button';
import { campaignStatusToAdminTone } from '@/lib/admin_kit';
import { StatusBadge } from '@/shell/status_badge';
import { ControlPlaneSelectionPanel } from '@/shell/control_plane_selection_panel';
import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
export type CampaignsSelectionPanelProps = {
  selectedCampaign: Campaign | undefined;
  bulkBusy?: boolean;
  exportBusy?: boolean;
  onClone: () => void;
  onPause: () => void;
  onResume: () => void;
  onArchive: () => void;
  onExportCsv: () => void;
  onExportBundles: () => void;
  onClearSelection: () => void;
};

export function CampaignsSelectionPanel({
  selectedCampaign,
  bulkBusy = false,
  exportBusy = false,
  onClone,
  onPause,
  onResume,
  onArchive,
  onExportCsv,
  onExportBundles,
  onClearSelection,
}: CampaignsSelectionPanelProps) {
  const busy = bulkBusy || exportBusy;
  const selectedTitle = selectedCampaign?.name ?? selectedCampaign?.id;

  return (
    <ControlPlaneSelectionPanel
      emptyHint="Select a campaign in the list to run actions here."
      meta={
        selectedCampaign ? (
          <StatusBadge
            label={selectedCampaign.status_label ?? selectedCampaign.status}
            tone={campaignStatusToAdminTone(selectedCampaign.status)}
          />
        ) : null
      }
      selectedTitle={selectedTitle}
      title="Campaign actions"
    >
      {selectedCampaign?.id ? (
        <PrimaryActionButton asChild disabled={busy}>
          <Link to={`/campaigns/${selectedCampaign.id}/edit`}>Edit</Link>
        </PrimaryActionButton>
      ) : null}
      <SecondaryActionButton disabled={busy} type="button" onClick={onClone}>
        Clone
      </SecondaryActionButton>
      <SecondaryActionButton disabled={busy} type="button" onClick={onPause}>
        Pause
      </SecondaryActionButton>
      <SecondaryActionButton disabled={busy} type="button" onClick={onResume}>
        Resume
      </SecondaryActionButton>
      <Button disabled={busy} type="button" variant="destructive" onClick={onArchive}>
        Archive
      </Button>
      <SecondaryActionButton disabled={busy} type="button" onClick={onExportCsv}>
        Export CSV
      </SecondaryActionButton>
      <SecondaryActionButton disabled={busy} type="button" onClick={onExportBundles}>
        Export JSON
      </SecondaryActionButton>
      <SecondaryActionButton disabled={busy} type="button" onClick={onClearSelection}>
        Clear selection
      </SecondaryActionButton>
    </ControlPlaneSelectionPanel>
  );
}
