import { useCallback, useState } from 'react';

import { applyLicense } from '@/api/platform_api';
import { PrimaryActionButton } from '@/shell/action_buttons';
import { ErrorBlock } from '@/shell/error_block';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { adminSpacing, adminTypography } from '@/lib/admin_kit';
import { toError } from '@/lib/admin_error';
import { requireNonEmpty } from '@/lib/admin_validation_error';
import { cn } from '@/lib/utils';
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
    const tokenResult = requireNonEmpty(draftToken, 'License token', 'token');
    if (!tokenResult.ok) {
      setError(tokenResult.error);
      return;
    }
    setApplying(true);
    setError(undefined);
    setSuccess(false);
    try {
      await applyLicense({ token: tokenResult.value });
      setSuccess(true);
      setDraftToken('');
      load.bumpStatusRefresh();
      onApplied?.();
    } catch (err: unknown) {
      setError(toError(err));
    } finally {
      setApplying(false);
    }
  }, [draftToken, load, onApplied]);

  return (
    <div className={`grid ${adminSpacing.gap.xl}`}>
      {showStatus && load.licenseStatus ? (
        <LicenseStatusSummary status={load.licenseStatus} />
      ) : null}
      {showStatus && load.statusError && !load.licenseStatus ? (
        <ErrorBlock title="Could not load license status" error={load.statusError} />
      ) : null}
      <div className={`grid ${adminSpacing.gap.md}`}>
        <Label htmlFor="license-token">{title}</Label>
        {description ? (
          <p className={cn('whitespace-normal', adminTypography.bodyMuted)}>{description}</p>
        ) : null}
        <Textarea
          id="license-token"
          className={cn('min-h-[7.5rem]', adminTypography.monoData)}
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
        <p className={adminTypography.bodyMuted}>License applied. Status refreshed.</p>
      ) : null}
      {error ? <ErrorBlock title="License apply failed" error={error} /> : null}
    </div>
  );
}

function LicenseStatusSummary({ status }: { status: LicenseStatus }) {
  return (
    <dl className={cn('grid', adminSpacing.gap.xs, adminTypography.body)}>
      <div>
        <dt className={adminTypography.labelMuted}>State</dt>
        <dd>{status.state ?? ''}</dd>
      </div>
      {status.valid_until ? (
        <div>
          <dt className={adminTypography.labelMuted}>Valid until</dt>
          <dd>{status.valid_until}</dd>
        </div>
      ) : null}
      {status.deployment_id ? (
        <div>
          <dt className={adminTypography.labelMuted}>Deployment</dt>
          <dd className={adminTypography.monoData}>{status.deployment_id}</dd>
        </div>
      ) : null}
    </dl>
  );
}
