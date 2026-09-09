import { useCallback, useState } from 'react';

import { applyLicense } from '@/api/platform_api';
import { PrimaryActionButton } from '@/shell/action_buttons';
import { ErrorBlock } from '@/shell/error_block';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import type { LicenseStatus } from '@/api/types';
import type { LicenseApplyFormLoad } from '@/domains/onboarding/use_license_apply_form_load';

export type LicenseApplyFormProps = {
  load: LicenseApplyFormLoad;
  title?: string;
  description?: string;
  onApplied?: () => void;
  showStatus?: boolean;
  textareaRows?: number;
};

export function LicenseApplyForm({
  load,
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
      load.bumpStatusRefresh();
      onApplied?.();
    } catch (err: unknown) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setApplying(false);
    }
  }, [draftToken, load, onApplied]);

  return (
    <div >
      {showStatus && load.licenseStatus ? (
        <LicenseStatusSummary status={load.licenseStatus} />
      ) : null}
      {showStatus && load.statusError && !load.licenseStatus ? (
        <ErrorBlock title="Could not load license status" message={load.statusError.message} />
      ) : null}
      <div >
        <Label htmlFor="license-token">{title}</Label>
        {description ? <p >{description}</p> : null}
        <Textarea
          id="license-token"
         
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
        <p >License applied. Status refreshed.</p>
      ) : null}
      {error ? <ErrorBlock title="License apply failed" message={error.message} /> : null}
    </div>
  );
}

function LicenseStatusSummary({ status }: { status: LicenseStatus }) {
  return (
    <dl >
      <div>
        <dt >State</dt>
        <dd>{status.state ?? ''}</dd>
      </div>
      {status.valid_until ? (
        <div>
          <dt >Valid until</dt>
          <dd>{status.valid_until}</dd>
        </div>
      ) : null}
      {status.deployment_id ? (
        <div>
          <dt >Deployment</dt>
          <dd >{status.deployment_id}</dd>
        </div>
      ) : null}
    </dl>
  );
}
