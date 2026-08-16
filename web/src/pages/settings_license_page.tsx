import { useEffect } from 'react';
import type { LicenseStatusDTO } from '../types/api/license.js';
import { surfaceServiceErrorToast } from '../helpers/service_error_toast.js';
import { useResource } from '../hooks/use_resource.js';
import { Breadcrumbs } from '../components/breadcrumbs.js';
import { ButtonLink } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { Icon } from '../components/icon.js';
import { StatusBadge } from '../components/status_badge.js';

/**
 * License status with link to apply on platform settings.
 */
export function SettingsLicensePage() {
  const { data, loading, error } = useResource<LicenseStatusDTO>('/api/v1/license/status');

  useEffect(() => {
    if (error) surfaceServiceErrorToast(error);
  }, [error]);

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
          On-prem deployment license. Apply a new JWT from{' '}
          <a href="/settings">Platform settings</a>.
        </p>
      </div>

      {loading ? <p className="text-muted">Loading…</p> : null}
      {error ? <ErrorBlock error={error} fallbackTitle="Failed to load license status" /> : null}
      {data ? (
        <section className="section-card stack" data-testid="license-status-panel">
          <dl className="definition-list">
            <dt>Deployment ID</dt>
            <dd className="font-mono">{data.deployment_id ?? '—'}</dd>
            <dt>State</dt>
            <dd>{data.state ? <StatusBadge status={data.state} /> : '—'}</dd>
            <dt>Valid until</dt>
            <dd>
              {data.valid_until ? new Date(data.valid_until).toLocaleString() : '—'}
            </dd>
          </dl>
          <ButtonLink
            label="Apply license on Platform settings"
            href="/settings"
            variant="secondary"
            size="sm"
            data-testid="license-apply-link"
          />
        </section>
      ) : null}
    </>
  );
}
