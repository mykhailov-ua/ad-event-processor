import { useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import type { LicenseStatus } from '../../helpers/settings_api.js';
import * as auth from '../../helpers/auth.js';
import { can } from '../../helpers/permissions.js';
import { Button } from '../system/button.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import { SettingsSubnav } from './settings_hub.js';
import styles from './settings_shared.module.css';

export type LicensePanelProps = {
  status: LicenseStatus | null;
  loading: boolean;
  error: unknown;
  applying: boolean;
  onApply: (token: string) => void;
};

export function LicensePanel({ status, loading, error, applying, onApply }: LicensePanelProps) {
  const permissions = auth.getUser()?.permissions ?? [];
  const canApply = can(permissions, 'settings:write');
  const [token, setToken] = useState('');

  if (error && !status) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load license status" />;
  }

  const onSubmit = (event: FormEvent) => {
    event.preventDefault();
    const trimmed = token.trim();
    if (!trimmed) return;
    onApply(trimmed);
    setToken('');
  };

  return (
    <div className={styles.root} data-testid="settings-license-page">
      <PageChrome
        title="License"
        badge={
          <Link to="/settings" className={styles.bannerLink}>
            Platform
          </Link>
        }
      />
      <SettingsSubnav />

      {loading && !status ? (
        <PageSkeleton rows={4} />
      ) : status ? (
        <div className={styles.content}>
          <div className={styles.kpiRow}>
            <div className={styles.kpiTile}>
              <p className={styles.kpiLabel}>State</p>
              <p className={styles.kpiValue}>{status.state ?? '-'}</p>
            </div>
            <div className={styles.kpiTile}>
              <p className={styles.kpiLabel}>Plan</p>
              <p className={styles.kpiValue}>{status.plan_code ?? '-'}</p>
            </div>
            <div className={styles.kpiTile}>
              <p className={styles.kpiLabel}>Valid until</p>
              <p className={styles.kpiValue}>{status.valid_until ?? '-'}</p>
            </div>
            <div className={styles.kpiTile}>
              <p className={styles.kpiLabel}>Days to expiry</p>
              <p className={styles.kpiValue}>
                {status.days_to_expiry != null ? String(status.days_to_expiry) : '-'}
              </p>
            </div>
          </div>
          <p className={styles.hint}>Deployment ID: {status.deployment_id ?? '-'}</p>
          <p className={styles.hint}>Host fingerprint: {status.host_fingerprint ?? '-'}</p>
          <p className={styles.hint}>HWID v2: {status.hwid_v2 ?? '-'}</p>
          {status.hwid_match != null ? (
            <p className={styles.hint}>HWID match: {status.hwid_match ? 'yes' : 'no'}</p>
          ) : null}
          {status.max_rps != null ? (
            <p className={styles.hint}>Max RPS: {String(status.max_rps)}</p>
          ) : null}
          {status.trial_self_serve_url ? (
            <p className={styles.hint}>
              Trial:{' '}
              <a href={status.trial_self_serve_url} target="_blank" rel="noreferrer">
                Self-serve
              </a>
            </p>
          ) : null}
          {status.support_url ? (
            <p className={styles.hint}>
              Support:{' '}
              <a href={status.support_url} target="_blank" rel="noreferrer">
                Contact
              </a>
            </p>
          ) : null}

          {canApply ? (
            <form className={styles.formStack} onSubmit={onSubmit}>
              <label className={styles.field}>
                <span className={styles.fieldLabel}>License JWT</span>
                <textarea
                  className={styles.textarea}
                  value={token}
                  onChange={(e) => setToken(e.target.value)}
                  placeholder="Paste license token (not shown after apply)"
                />
              </label>
              <Button type="submit" variant="primary" disabled={applying || !token.trim()}>
                Apply license
              </Button>
            </form>
          ) : (
            <p className={styles.hint}>settings:write required to apply a new license token.</p>
          )}
        </div>
      ) : null}
    </div>
  );
}
