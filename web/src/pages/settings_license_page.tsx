import { useEffect, useState } from 'react';
import type { LicenseStatusDTO } from '../types/api/license.js';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { apiConfirmed } from '../helpers/confirmed_api.js';
import { to } from '../lib/to.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { surfaceServiceErrorToast } from '../helpers/service_error_toast.js';
import { useResource } from '../hooks/use_resource.js';
import { Breadcrumbs } from '../components/breadcrumbs.js';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { Icon } from '../components/icon.js';
import { StatusBadge } from '../components/status_badge.js';
import { AlertBanner } from '../components/alert_banner.js';

/**
 * License status, host identity for vendor renewal, and JWT apply.
 */
export function SettingsLicensePage() {
  const user = auth.getUser();
  const canWrite = can(user?.permissions ?? [], 'settings:write');
  const { data, loading, error, reload } = useResource<LicenseStatusDTO>('/api/v1/license/status');
  const [licenseTokenInput, setLicenseTokenInput] = useState('');
  const [applying, setApplying] = useState(false);

  useEffect(() => {
    if (error) surfaceServiceErrorToast(error);
  }, [error]);

  const handleLicenseApply = async () => {
    if (!canWrite) return;
    const token = licenseTokenInput.trim();
    if (!token) {
      pushToastMessage({ title: 'License', message: 'Paste the license JWT from your vendor' });
      return;
    }
    setApplying(true);
    const [res, err] = await to(apiConfirmed('/api/v1/license/apply', {
      method: 'POST',
      body: JSON.stringify({ token }),
    }));
    if (err) {
      if (err instanceof ConfirmCancelledError) {
        setApplying(false);
        return;
      }
      const v = mapServiceError(err);
      pushToastMessage({ title: v.title, message: v.message, code: v.code });
      setApplying(false);
      return;
    }
    setLicenseTokenInput('');
    const body = (res?.data ?? {}) as LicenseStatusDTO;
    const until = body.valid_until ?? '';
    pushToastMessage({
      title: 'License updated',
      message: until ? `Valid until ${until}` : 'License applied',
    });
    reload();
    setApplying(false);
  };

  const bindMismatch = data?.hwid_match === false;

  return (
    <>
      <div className="page-header">
        <Breadcrumbs items={[
          { label: 'Settings', href: '/settings' },
          { label: 'License' },
        ]}
        />
        <div className="page-header__row">
          <div className="flex items-center gap-2">
            <Icon name="key" size={20} className="text-muted" />
            <h1 className="page-header__title">License</h1>
          </div>
        </div>
        <p className="text-muted text-sm">
          On-prem deployment license. Paste the monthly JWT from your vendor; include{' '}
          <span className="font-mono">hwid_v2</span> or host fingerprint when requesting renewal.
        </p>
      </div>

      {loading ? <p className="text-muted">Loading…</p> : null}
      {error ? <ErrorBlock error={error} fallbackTitle="Failed to load license status" /> : null}
      {bindMismatch ? (
        <AlertBanner
          variant="error"
          message="Host identity does not match the bound license. Renewal JWT must match this deployment."
        />
      ) : null}
      {data ? (
        <section className="section-card stack" data-testid="license-status-panel">
          <dl className="definition-list">
            <dt>Deployment ID</dt>
            <dd className="font-mono">{data.deployment_id || '—'}</dd>
            <dt>State</dt>
            <dd>{data.state ? <StatusBadge status={data.state} /> : '—'}</dd>
            <dt>Valid until</dt>
            <dd>
              {data.valid_until ? new Date(data.valid_until).toLocaleString() : '—'}
            </dd>
            {data.days_to_expiry != null && data.days_to_expiry > 0 ? (
              <>
                <dt>Days to expiry</dt>
                <dd>{data.days_to_expiry}</dd>
              </>
            ) : null}
            <dt>Host fingerprint</dt>
            <dd className="font-mono break-all" data-testid="license-host-fingerprint">
              {data.host_fingerprint || '—'}
            </dd>
            <dt>HWID v2</dt>
            <dd className="font-mono break-all" data-testid="license-hwid-v2">
              {data.hwid_v2 || '—'}
            </dd>
            {data.hwid_match != null ? (
              <>
                <dt>Bind match</dt>
                <dd data-testid="license-hwid-match">
                  {data.hwid_match ? 'Match' : 'Mismatch'}
                </dd>
              </>
            ) : null}
          </dl>
        </section>
      ) : null}

      <section className="section-card stack" data-testid="license-apply-panel">
        <h2 className="section-card__title">License renewal</h2>
        <p className="text-muted text-sm">
          Offline on-prem: paste the monthly JWT from your vendor. No outbound license server ping.
        </p>
        {!canWrite ? (
          <AlertBanner variant="info" message="You need settings:write to apply a license JWT." />
        ) : (
          <>
            <label className="form-field">
              <span className="form-field__label">License JWT</span>
              <textarea
                className="form-field__input font-mono"
                rows={4}
                value={licenseTokenInput}
                placeholder="Paste full JWT line from vendor"
                onChange={(e) => setLicenseTokenInput(e.target.value)}
                data-testid="license-token-input"
              />
            </label>
            <Button
              label="Apply license"
              variant="primary"
              size="sm"
              disabled={applying}
              onClick={() => void handleLicenseApply()}
              data-testid="license-apply-button"
            />
          </>
        )}
      </section>
    </>
  );
}
