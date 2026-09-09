import type { AuditLog } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { ControlPlaneSelectionPanel } from '@/shell/control_plane_selection_panel';
import { SecondaryActionButton } from '@/shell/action_buttons';

export type AuditSelectionPanelProps = {
  selectedEntry: AuditLog | undefined;
  onClearSelection: () => void;
};

function auditMetadataFields(metadata: AuditLog['metadata']): {
  authSource?: string;
  apiKeyId?: string;
} {
  if (!metadata || typeof metadata !== 'object' || Array.isArray(metadata)) {
    return {};
  }
  const record = metadata as Record<string, unknown>;
  const authSource = typeof record.auth_source === 'string' ? record.auth_source : undefined;
  const apiKeyId = typeof record.api_key_id === 'string' ? record.api_key_id : undefined;
  return { authSource, apiKeyId };
}

export function AuditSelectionPanel({
  selectedEntry,
  onClearSelection,
}: AuditSelectionPanelProps) {
  const metadata = selectedEntry ? auditMetadataFields(selectedEntry.metadata) : {};
  const selectedTitle = selectedEntry
    ? `${selectedEntry.action ?? 'action'} · ${selectedEntry.target_type ?? 'target'}`
    : undefined;

  return (
    <ControlPlaneSelectionPanel
      emptyHint="Select an audit entry to review details."
      meta={
        selectedEntry ? (
          <>
            <div>Admin: {selectedEntry.admin_id ?? ''}</div>
            <div>Target ID: {selectedEntry.target_id ?? ''}</div>
            <div>
              Time:{' '}
              {displayTimestamp(selectedEntry.created_at, selectedEntry.created_at_display)}
            </div>
            {metadata.authSource ? <div>Auth: {metadata.authSource}</div> : null}
            {metadata.apiKeyId ? <div>API key: {metadata.apiKeyId}</div> : null}
            {selectedEntry.is_masked ? <div>PII masked in export</div> : null}
          </>
        ) : null
      }
      selectedTitle={selectedTitle}
      title="Audit entry"
    >
      <SecondaryActionButton type="button" onClick={onClearSelection}>
        Clear selection
      </SecondaryActionButton>
    </ControlPlaneSelectionPanel>
  );
}
