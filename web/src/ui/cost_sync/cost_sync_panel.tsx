import { useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import * as auth from '../../helpers/auth.js';
import type {
  CostSyncCredential,
  CostSyncNetworkSchema,
  CostSyncRun,
} from '../../helpers/integrations_api.js';
import { can, canReadCampaigns } from '../../helpers/permissions.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { PaginationBar } from '../system/pagination_bar.js';
import { CustomerScopeBar } from '../integrations/customer_scope_bar.js';
import shared from '../integrations/integrations_shared.module.css';
import styles from './cost_sync.module.css';

export type CostSyncPanelProps = {
  customerId: string;
  networks: CostSyncNetworkSchema[];
  credentials: CostSyncCredential[];
  history: CostSyncRun[];
  historyLimit: number;
  historyOffset: number;
  historyHasMore: boolean;
  loading: boolean;
  error: unknown;
  busyNetwork: string | null;
  onCustomerApply: (customerId: string) => void;
  onHistoryOffsetChange: (offset: number) => void;
  onSaveCredential: (network: string, accountId: string, apiKey: string) => void;
  onDeleteCredential: (network: string) => void;
  onRunSync: (network: string) => void;
};

export function CostSyncPanel({
  customerId,
  networks,
  credentials,
  history,
  historyLimit,
  historyOffset,
  historyHasMore,
  loading,
  error,
  busyNetwork,
  onCustomerApply,
  onHistoryOffsetChange,
  onSaveCredential,
  onDeleteCredential,
  onRunSync,
}: CostSyncPanelProps) {
  const permissions = auth.getUser()?.permissions ?? [];
  const canWrite = can(permissions, 'campaigns:write');
  const canList = canReadCampaigns(permissions);

  const credentialByNetwork = useMemo(() => {
    const map = new Map<string, CostSyncCredential>();
    for (const row of credentials) {
      if (row.network) map.set(row.network, row);
    }
    return map;
  }, [credentials]);

  const [draftAccount, setDraftAccount] = useState<Record<string, string>>({});
  const [draftApiKey, setDraftApiKey] = useState<Record<string, string>>({});

  if (!canList) {
    return <ErrorBlock error={new Error('forbidden')} fallbackTitle="Cost sync access denied" />;
  }

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load cost sync" />;
  }

  const historyTotal = historyHasMore
    ? historyOffset + history.length + historyLimit
    : historyOffset + history.length;

  return (
    <div className={shared.panelRoot} data-testid="cost-sync-view">
      <PageChrome
        title="Cost sync"
        badge={
          <Link to="/integrations" className={shared.bannerLink}>
            All integrations
          </Link>
        }
      />
      <CustomerScopeBar customerId={customerId} onApply={onCustomerApply} />

      {!customerId ? (
        <div className={shared.hint}>Enter a customer ID and apply to load credentials.</div>
      ) : (
        <>
          <section className={styles.section}>
            <h2 className={shared.sectionTitle}>Networks</h2>
            <div className={`${shared.gridTable} ${shared.gridTableSubgrid}`} role="grid">
              <div className={`${shared.gridHeader} ${styles.colsNetworks}`} role="row">
                <span className={shared.gridCell} role="columnheader">
                  Network
                </span>
                <span className={shared.gridCell} role="columnheader">
                  Label
                </span>
                <span className={shared.gridCell} role="columnheader">
                  Credential
                </span>
              </div>
              {networks.length === 0 && !loading ? (
                <EmptyState message="No networks returned by API." />
              ) : (
                networks.map((network) => {
                  const key = network.network ?? '';
                  const cred = credentialByNetwork.get(key);
                  return (
                    <div
                      key={key}
                      className={`${shared.gridRow} ${styles.colsNetworks}`}
                      role="row"
                    >
                      <span className={shared.gridCell} role="gridcell">
                        {key}
                      </span>
                      <span className={shared.gridCell} role="gridcell">
                        {network.label ?? key}
                      </span>
                      <span className={shared.gridCell} role="gridcell">
                        {cred ? 'configured' : 'missing'}
                      </span>
                    </div>
                  );
                })
              )}
            </div>
          </section>

          <section className={styles.section}>
            <h2 className={shared.sectionTitle}>Credentials</h2>
            {networks.map((network) => {
              const key = network.network ?? '';
              const cred = credentialByNetwork.get(key);
              return (
                <div key={key} className={shared.formStack}>
                  <h3 className={shared.sectionTitle}>{network.label ?? key}</h3>
                  <label className={shared.field}>
                    <span className={shared.fieldLabel}>
                      {network.account_id_label ?? 'Account ID'}
                    </span>
                    <input
                      className={shared.textInput}
                      value={draftAccount[key] ?? cred?.account_id ?? ''}
                      onChange={(event) =>
                        setDraftAccount((prev) => ({ ...prev, [key]: event.target.value }))
                      }
                      disabled={!canWrite}
                    />
                  </label>
                  <label className={shared.field}>
                    <span className={shared.fieldLabel}>API key / token (blank keeps stored)</span>
                    <input
                      className={shared.textInput}
                      type="password"
                      autoComplete="off"
                      value={draftApiKey[key] ?? ''}
                      onChange={(event) =>
                        setDraftApiKey((prev) => ({ ...prev, [key]: event.target.value }))
                      }
                      disabled={!canWrite}
                    />
                  </label>
                  {canWrite ? (
                    <div className={shared.actions}>
                      <Button
                       
                        variant="primary"
                        disabled={busyNetwork === key}
                        onClick={() =>
                          onSaveCredential(
                            key,
                            draftAccount[key] ?? cred?.account_id ?? '',
                            draftApiKey[key] ?? ''
                          )
                        }
                      >
                        Save
                      </Button>
                      {cred ? (
                        <Button
                         
                          variant="secondary"
                          disabled={busyNetwork === key}
                          onClick={() => onDeleteCredential(key)}
                        >
                          Delete
                        </Button>
                      ) : null}
                      <Button
                       
                        variant="secondary"
                        disabled={busyNetwork === key}
                        onClick={() => onRunSync(key)}
                      >
                        Run sync
                      </Button>
                    </div>
                  ) : null}
                </div>
              );
            })}
          </section>

          <section className={styles.section}>
            <h2 className={shared.sectionTitle}>History</h2>
            <div className={shared.content}>
              <div className={shared.gridTable} role="grid">
                <div className={`${shared.gridHeader} ${styles.colsHistory}`} role="row">
                  <span className={shared.gridCell} role="columnheader">
                    ID
                  </span>
                  <span className={shared.gridCell} role="columnheader">
                    Network
                  </span>
                  <span className={shared.gridCell} role="columnheader">
                    Date
                  </span>
                  <span className={shared.gridCell} role="columnheader">
                    Status
                  </span>
                  <span className={shared.gridCell} role="columnheader">
                    Rows
                  </span>
                  <span className={shared.gridCell} role="columnheader">
                    Error
                  </span>
                </div>
                {history.length === 0 && !loading ? (
                  <EmptyState message="No sync runs for this scope." />
                ) : (
                  history.map((row) => (
                    <div
                      key={String(row.id)}
                      className={`${shared.gridRow} ${styles.colsHistory}`}
                      role="row"
                    >
                      <span className={shared.gridCell} role="gridcell">
                        {row.id}
                      </span>
                      <span className={shared.gridCell} role="gridcell">
                        {row.network}
                      </span>
                      <span className={shared.gridCell} role="gridcell">
                        {row.cost_date}
                      </span>
                      <span className={shared.gridCell} role="gridcell">
                        {row.status}
                      </span>
                      <span className={shared.gridCell} role="gridcell">
                        {row.rows_imported}
                      </span>
                      <span className={shared.gridCell} role="gridcell">
                        {row.error_message ?? ''}
                      </span>
                    </div>
                  ))
                )}
              </div>
            </div>
            <PaginationBar
              limit={historyLimit}
              offset={historyOffset}
              total={historyTotal}
              onOffsetChange={onHistoryOffsetChange}
            />
          </section>
        </>
      )}
    </div>
  );
}
