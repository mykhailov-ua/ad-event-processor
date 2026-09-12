import { Link } from 'react-router-dom';

import type { Campaign } from '@/api/types';
import { Button } from '@/components/ui/button';
import { campaignStatusToAdminTone } from '@/lib/admin_kit';
import { formatCampaignStatusLabel } from '@/lib/admin_typography';
import type { AdminValidationError } from '@/lib/admin_validation_error';
import { AdminMutationError } from '@/shell/admin_error';
import { StatusBadge } from '@/shell/status_badge';
import { ControlPlaneSelectionPanel } from '@/shell/control_plane_selection_panel';
import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import { ValidationErrorBlock } from '@/shell/validation_error_block';

export type CampaignsSelectionPanelProps = {
  selectedCampaign: Campaign | undefined;
  selectedCount?: number;
  bulkActionError?: Error;
  bulkBusy?: boolean;
  exportError?: Error;
  exportBusy?: boolean;
  selectionGuardError?: AdminValidationError;
  variant?: 'sidebar' | 'toolbar' | 'inline';
  onClone: () => void;
  onBulkEdit: () => void;
  onPause: () => void;
  onResume: () => void;
  onArchive: () => void;
  onExportCsv: () => void;
  onExportBundles: () => void;
  onClearSelection: () => void;
};

const actionButtonShape = 'default' as const;

export function CampaignsSelectionPanel({
  selectedCampaign,
  selectedCount = 0,
  bulkActionError,
  bulkBusy = false,
  exportError,
  exportBusy = false,
  selectionGuardError,
  variant = 'sidebar',
  onClone,
  onBulkEdit,
  onPause,
  onResume,
  onArchive,
  onExportCsv,
  onExportBundles,
  onClearSelection,
}: CampaignsSelectionPanelProps) {
  const busy = bulkBusy || exportBusy;
  const selectedTitle = selectedCampaign?.name ?? selectedCampaign?.id;
  const isInline = variant === 'inline';

  const statusBadge = selectedCampaign ? (
    <StatusBadge
      label={formatCampaignStatusLabel(selectedCampaign.status)}
      size={isInline ? 'control' : 'default'}
      tone={campaignStatusToAdminTone(selectedCampaign.status)}
    />
  ) : null;

  const actionButtons = (
    <>
      {selectedCampaign?.id ? (
        <PrimaryActionButton asChild disabled={busy} shape={actionButtonShape}>
          <Link to={`/campaigns/${selectedCampaign.id}/edit`}>Edit</Link>
        </PrimaryActionButton>
      ) : null}
      <SecondaryActionButton
        disabled={busy || selectedCount === 0}
        shape={actionButtonShape}
        type="button"
        onClick={onBulkEdit}
      >
        Bulk edit{selectedCount > 0 ? ` (${selectedCount})` : ''}
      </SecondaryActionButton>
      <SecondaryActionButton
        disabled={busy || !selectedCampaign?.id}
        shape={actionButtonShape}
        type="button"
        onClick={onClone}
      >
        Clone
      </SecondaryActionButton>
      <SecondaryActionButton
        disabled={busy}
        shape={actionButtonShape}
        type="button"
        onClick={onPause}
      >
        Pause
      </SecondaryActionButton>
      <SecondaryActionButton
        disabled={busy}
        shape={actionButtonShape}
        type="button"
        onClick={onResume}
      >
        Resume
      </SecondaryActionButton>
      <Button
        disabled={busy}
        shape={actionButtonShape}
        type="button"
        variant="destructive"
        onClick={onArchive}
      >
        Archive
      </Button>
      <SecondaryActionButton
        disabled={busy}
        shape={actionButtonShape}
        type="button"
        onClick={onExportCsv}
      >
        Export CSV
      </SecondaryActionButton>
      <SecondaryActionButton
        disabled={busy}
        shape={actionButtonShape}
        type="button"
        onClick={onExportBundles}
      >
        Export JSON
      </SecondaryActionButton>
      <SecondaryActionButton
        disabled={busy}
        shape={actionButtonShape}
        type="button"
        onClick={onClearSelection}
      >
        Clear selection
      </SecondaryActionButton>
    </>
  );

  const errors = (
    <>
      {selectionGuardError ? (
        <ValidationErrorBlock error={selectionGuardError} title="Select a campaign first" />
      ) : null}
      {bulkActionError ? (
        <AdminMutationError error={bulkActionError} title="Bulk action failed" />
      ) : null}
      {exportError ? <AdminMutationError error={exportError} title="Export failed" /> : null}
    </>
  );

  if (isInline) {
    return (
      <>
        {errors}
        {statusBadge}
        {actionButtons}
      </>
    );
  }

  return (
    <ControlPlaneSelectionPanel
      emptyHint="Select a campaign in the list to run actions here."
      meta={statusBadge}
      selectedTitle={selectedTitle}
      title="Campaign actions"
      variant={variant}
    >
      {errors}
      {actionButtons}
    </ControlPlaneSelectionPanel>
  );
}
