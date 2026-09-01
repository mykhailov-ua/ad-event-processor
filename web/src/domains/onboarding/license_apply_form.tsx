import { useCallback, useState } from 'react';

import { applyLicense, getLicenseStatus } from '@/api/platform_api';
import { PrimaryActionButton } from '@/components/system/action_buttons';
import { ErrorBlock } from '@/components/system/error_block';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import type { LicenseStatus } from '@/api/types';
import { useResource } from '@/hooks/use_resource';

export type LicenseApplyFormProps = {
  title?: string;
  description?: string;
  onApplied?: () => void;
  showStatus?: boolean;
  textareaRows?: number;
};

export function LicenseApplyForm({
  title = 'License JWT',
  description = 'Paste the license token from your deployment bundle or vendor portal.',
  onApplied,
  showStatus = true,
  textareaRows = 5,
}: LicenseApplyFormProps) {
  const [draftToken, setDraftToken] = useState('');
  const [applying, setApplying] = useState(false);
  const [error, setError] = useState<Error | undefined>();
  const [success, setSuccess] = useState(false);
  const [refreshToken, setRefreshToken] = useState(0);

  const { data: licenseStatus } = useResource(
    (signal) => {
      if (!showStatus) {
        return Promise.resolve(undefined);
      }
      return getLicenseStatus(signal);
    },
    [showStatus, refreshToken],
  );

  const onApply = useCallback(async () => {
    const token = draftToken.trim();
    if (!token) {
      return;
    }
    setApplying(true);
    setError(undefined);
    setSuccess(false);
    try {
      await applyLicense({ token });
      setSuccess(true);
      setDraftToken('');
      setRefreshToken((value) => value + 1);
      onApplied?.();
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setApplying(false);
    }
  }, [draftToken, onApplied]);

  return (
    <div className="grid gap-4">
      {showStatus && licenseStatus ? <LicenseStatusSummary status={licenseStatus} /> : null}
      <div className="grid gap-2">
        <Label htmlFor="license-token">{title}</Label>
        {description ? <p className="text-sm text-muted-foreground">{description}</p> : null}
        <Textarea
          id="license-token"
          className="min-h-[7.5rem] font-mono text-xs"
          rows={textareaRows}
          value={draftToken}
          onChange={(event) => setDraftToken(event.target.value)}
        />
      </div>
      <div>
        <PrimaryActionButton
          disabled={!draftToken.trim()}
          loading={applying}
          onClick={() => void onApply()}
          type="button"
        >
          Apply license
        </PrimaryActionButton>
      </div>
      {success ? (
        <p className="text-sm text-muted-foreground">License applied. Status refreshed.</p>
      ) : null}
      {error ? <ErrorBlock title="License apply failed" message={error.message} /> : null}
    </div>
  );
}

function LicenseStatusSummary({ status }: { status: LicenseStatus }) {
  return (
    <dl className="grid gap-1 text-sm">
      <div>
        <dt className="text-muted-foreground">State</dt>
        <dd>{status.state ?? ''}</dd>
      </div>
      {status.valid_until ? (
        <div>
          <dt className="text-muted-foreground">Valid until</dt>
          <dd>{status.valid_until}</dd>
        </div>
      ) : null}
      {status.deployment_id ? (
        <div>
          <dt className="text-muted-foreground">Deployment</dt>
          <dd className="font-mono text-xs">{status.deployment_id}</dd>
        </div>
      ) : null}
    </dl>
  );
}
