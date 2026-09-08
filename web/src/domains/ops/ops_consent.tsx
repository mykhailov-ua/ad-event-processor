import type { OpsConsentProofsResponse } from '@/api/types';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsPageBlockingError, OpsPageLoading, OpsPageShell } from '@/domains/ops/ops_page_shell';
import { FILTER_PANEL_NARROW_CLASS } from '@/shell/filter_panel';
import { ErrorBlock } from '@/shell/error_block';
import { JsonPayloadView } from '@/shell/json_payload_view';

export type OpsConsentProps = {
  payload: OpsConsentProofsResponse | undefined;
  fetching: boolean;
  listRevalidating?: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  draftUserId: string;
  draftPurposes: string;
  draftSource: string;
  draftTimestamp: string;
  draftSigningSecret: string;
  draftSignatureHex: string;
  recording: boolean;
  recordError: Error | undefined;
  recordSuccess: boolean;
  onDraftUserIdChange: (value: string) => void;
  onDraftPurposesChange: (value: string) => void;
  onDraftSourceChange: (value: string) => void;
  onDraftTimestampChange: (value: string) => void;
  onDraftSigningSecretChange: (value: string) => void;
  onDraftSignatureHexChange: (value: string) => void;
  onRecordConsent: () => void;
};

export function OpsConsent({
  payload,
  fetching,
  listRevalidating = false,
  error,
  hasSnapshot,
  draftUserId,
  draftPurposes,
  draftSource,
  draftTimestamp,
  draftSigningSecret,
  draftSignatureHex,
  recording,
  recordError,
  recordSuccess,
  onDraftUserIdChange,
  onDraftPurposesChange,
  onDraftSourceChange,
  onDraftTimestampChange,
  onDraftSigningSecretChange,
  onDraftSignatureHexChange,
  onRecordConsent,
}: OpsConsentProps) {
  if (fetching && !hasSnapshot && !error) {
    return <OpsPageLoading />;
  }

  if (error && !hasSnapshot) {
    return (
      <OpsPageBlockingError
        error={error}
        pageTitle="Consent proofs"
        title="Could not load consent proofs"
      />
    );
  }

  return (
    <OpsPageShell title="Consent proofs">
      <section className={FILTER_PANEL_NARROW_CLASS}>
        <h2 className="text-sm font-medium">Record signed consent</h2>
        <p className="text-sm text-muted-foreground">
          POST /api/v1/consent with X-Consent-Signature. Use a local HMAC secret (test) or paste a
          precomputed signature hex.
        </p>
        <div className="grid gap-2">
          <Label htmlFor="consent-user-id">User ID</Label>
          <Input
            id="consent-user-id"
            value={draftUserId}
            onChange={(event) => onDraftUserIdChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="consent-purposes">Purposes (int32 bitmask)</Label>
          <Input
            id="consent-purposes"
            inputMode="numeric"
            value={draftPurposes}
            onChange={(event) => onDraftPurposesChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="consent-source">Source</Label>
          <Input
            id="consent-source"
            value={draftSource}
            onChange={(event) => onDraftSourceChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="consent-timestamp">Timestamp (optional, RFC3339)</Label>
          <Input
            id="consent-timestamp"
            value={draftTimestamp}
            onChange={(event) => onDraftTimestampChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="consent-hmac-secret">HMAC secret (local test)</Label>
          <Input
            autoComplete="off"
            id="consent-hmac-secret"
            type="password"
            value={draftSigningSecret}
            onChange={(event) => onDraftSigningSecretChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="consent-signature">Signature hex (optional if secret set)</Label>
          <Input
            id="consent-signature"
            value={draftSignatureHex}
            onChange={(event) => onDraftSignatureHexChange(event.target.value)}
          />
        </div>
        <Button
          disabled={recording || !draftUserId.trim() || !draftSource.trim()}
          type="button"
          onClick={onRecordConsent}
        >
          {recording ? 'Recording...' : 'Record consent'}
        </Button>
        {recordSuccess ? (
          <p className="text-sm text-muted-foreground">Consent accepted by server.</p>
        ) : null}
        {recordError ? <ErrorBlock title="Record failed" message={recordError.message} /> : null}
      </section>

      {payload ? (
        <JsonPayloadView payload={payload} />
      ) : (
        <p className="text-muted-foreground">No consent proof payload returned.</p>
      )}
      {listRevalidating ? (
        <p className="text-sm text-muted-foreground">Refreshing proofs...</p>
      ) : null}
      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </OpsPageShell>
  );
}
