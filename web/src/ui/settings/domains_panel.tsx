import { useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import type { DomainHealth } from '../../helpers/domains_api.js';
import * as auth from '../../helpers/auth.js';
import { can } from '../../helpers/permissions.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import { SettingsSubnav } from './settings_hub.js';
import styles from './settings_shared.module.css';

export type DomainsPanelProps = {
  domains: DomainHealth[];
  loading: boolean;
  error: unknown;
  busyHostname: string | null;
  onAdd: (hostname: string) => void;
  onDelete: (hostname: string) => void;
  onProbe: (hostname: string) => void;
  onSslSetup: (hostname: string) => void;
  onPark: (body: { domain: string; cloudflare_zone_id: string }) => void;
};

export function DomainsPanel({
  domains,
  loading,
  error,
  busyHostname,
  onAdd,
  onDelete,
  onProbe,
  onSslSetup,
  onPark,
}: DomainsPanelProps) {
  const permissions = auth.getUser()?.permissions ?? [];
  const canWrite = can(permissions, 'settings:write');

  const [hostname, setHostname] = useState('');
  const [parkDomain, setParkDomain] = useState('');
  const [parkZoneId, setParkZoneId] = useState('');

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load domains" />;
  }

  const onAddSubmit = (event: FormEvent) => {
    event.preventDefault();
    const trimmed = hostname.trim();
    if (!trimmed) return;
    onAdd(trimmed);
    setHostname('');
  };

  const onParkSubmit = (event: FormEvent) => {
    event.preventDefault();
    const domain = parkDomain.trim();
    const zone = parkZoneId.trim();
    if (!domain || !zone) return;
    onPark({ domain, cloudflare_zone_id: zone });
    setParkDomain('');
    setParkZoneId('');
  };

  return (
    <div className={styles.root} data-testid="settings-domains-page">
      <PageChrome
        title="Domains"
        badge={
          <Link to="/settings" className={styles.bannerLink}>
            Platform
          </Link>
        }
      />
      <SettingsSubnav />

      {canWrite ? (
        <form className={styles.formStack} onSubmit={onAddSubmit}>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>Add hostname</span>
            <input
              className={styles.textInput}
              value={hostname}
              onChange={(e) => setHostname(e.target.value)}
              placeholder="trk.example.com"
            />
          </label>
          <Button type="submit" variant="primary" disabled={!hostname.trim()}>
            Add domain
          </Button>
        </form>
      ) : null}

      <div className={styles.content}>
        {loading && domains.length === 0 ? (
          <PageSkeleton rows={4} columns={6} />
        ) : domains.length === 0 ? (
          <EmptyState message="No custom domains registered." />
        ) : (
          <div className={`${styles.gridTable} ${styles.colsDomains}`} role="grid">
            <div className={styles.gridHeader} role="row">
              <span className={styles.gridCell} role="columnheader">
                Hostname
              </span>
              <span className={styles.gridCell} role="columnheader">
                Role
              </span>
              <span className={styles.gridCell} role="columnheader">
                Health
              </span>
              <span className={styles.gridCell} role="columnheader">
                SSL
              </span>
              <span className={styles.gridCell} role="columnheader">
                Detail
              </span>
              <span className={styles.gridCell} role="columnheader">
                Probed
              </span>
              <span className={styles.gridCell} role="columnheader">
                Actions
              </span>
            </div>
            {domains.map((row) => {
              const host = row.hostname ?? '';
              const busy = busyHostname === host;
              return (
                <div key={host} className={styles.gridRow} role="row">
                  <span className={styles.gridCell} role="gridcell">
                    {host || '-'}
                  </span>
                  <span className={styles.gridCell} role="gridcell">
                    {row.role ?? '-'}
                  </span>
                  <span className={styles.gridCell} role="gridcell">
                    {row.health_status ?? '-'}
                  </span>
                  <span className={styles.gridCell} role="gridcell">
                    {row.ssl_status ?? '-'}
                  </span>
                  <span className={styles.gridCell} role="gridcell">
                    {row.probe_detail ?? '-'}
                  </span>
                  <span className={styles.gridCell} role="gridcell">
                    {row.last_probe_at ?? '-'}
                  </span>
                  <span className={`${styles.gridCell} ${styles.actions}`} role="gridcell">
                    {canWrite && host ? (
                      <>
                        <Button
                          type="button"
                          size="sm"
                          disabled={busy}
                          onClick={() => onProbe(host)}
                        >
                          Probe
                        </Button>
                        <Button
                          type="button"
                          size="sm"
                          disabled={busy}
                          onClick={() => onSslSetup(host)}
                        >
                          SSL
                        </Button>
                        <Button
                          type="button"
                          size="sm"
                          variant="danger"
                          disabled={busy}
                          onClick={() => onDelete(host)}
                        >
                          Delete
                        </Button>
                      </>
                    ) : (
                      '-'
                    )}
                  </span>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {canWrite ? (
        <form className={styles.formStack} onSubmit={onParkSubmit}>
          <h3 className={styles.sectionTitle}>Park domain (Cloudflare)</h3>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>Domain</span>
            <input
              className={styles.textInput}
              value={parkDomain}
              onChange={(e) => setParkDomain(e.target.value)}
            />
          </label>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>Cloudflare zone ID</span>
            <input
              className={styles.textInput}
              value={parkZoneId}
              onChange={(e) => setParkZoneId(e.target.value)}
            />
          </label>
          <Button type="submit" variant="primary" disabled={!parkDomain.trim() || !parkZoneId.trim()}>
            Park domain
          </Button>
        </form>
      ) : null}
    </div>
  );
}
