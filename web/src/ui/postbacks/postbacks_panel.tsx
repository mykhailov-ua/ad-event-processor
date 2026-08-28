import { Link } from 'react-router-dom';
import * as auth from '../../helpers/auth.js';
import type { PostbackCampaignStatus, PostbackDlqRow } from '../../helpers/integrations_api.js';
import { can, canReadCampaigns } from '../../helpers/permissions.js';
import { useGridRowAction } from '../../helpers/use_grid_row_action.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { StubBanner } from '../system/stub_banner.js';
import shared from '../integrations/integrations_shared.module.css';
import styles from './postbacks.module.css';

export type PostbacksPanelProps = {
  campaignStatus: PostbackCampaignStatus[];
  dlq: PostbackDlqRow[];
  loading: boolean;
  error: unknown;
  retryBusyId: number | null;
  onRetryDlq: (id: number) => void;
};

export function PostbacksPanel({
  campaignStatus,
  dlq,
  loading,
  error,
  retryBusyId,
  onRetryDlq,
}: PostbacksPanelProps) {
  const permissions = auth.getUser()?.permissions ?? [];
  const onRetryDlqClick = useGridRowAction((id) => onRetryDlq(Number.parseInt(id, 10)));
  const canWrite = can(permissions, 'campaigns:write');
  const canList = canReadCampaigns(permissions);

  if (!canList) {
    return <ErrorBlock error={new Error('forbidden')} fallbackTitle="Postbacks access denied" />;
  }

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load postbacks" />;
  }

  return (
    <div className={shared.panelRoot} data-testid="integrations-postbacks-page">
      <PageChrome
        title="Postbacks"
        badge={
          <Link to="/integrations" className={shared.bannerLink}>
            All integrations
          </Link>
        }
      />
      <StubBanner
        title="Campaign CAPI configuration"
        message="Per-campaign postback and CAPI settings live on campaign detail under the Postbacks tab."
      />
      <p className={shared.hint}>
        Open a campaign from the{' '}
        <Link to="/campaigns" className={shared.bannerLink}>
          campaigns directory
        </Link>{' '}
        to edit postback config.
      </p>

      <section>
        <h2 className={shared.sectionTitle}>Campaign status</h2>
        <div className={shared.gridTable} role="grid">
          <div className={`${shared.gridHeader} ${styles.colsStatus}`} role="row">
            <span className={shared.gridCell} role="columnheader">
              Campaign
            </span>
            <span className={shared.gridCell} role="columnheader">
              Provider
            </span>
            <span className={shared.gridCell} role="columnheader">
              Last success
            </span>
            <span className={shared.gridCell} role="columnheader">
              DLQ pending
            </span>
          </div>
          {campaignStatus.length === 0 && !loading ? (
            <EmptyState message="No postback campaign status rows." />
          ) : (
            campaignStatus.map((row) => (
              <div
                key={row.campaign_id ?? row.provider}
                className={`${shared.gridRow} ${styles.colsStatus}`}
                role="row"
              >
                <span className={shared.gridCell} role="gridcell">
                  {row.campaign_id ? (
                    <Link to={`/campaigns/${row.campaign_id}`} className={shared.bannerLink}>
                      {row.campaign_id}
                    </Link>
                  ) : (
                    '-'
                  )}
                </span>
                <span className={shared.gridCell} role="gridcell">
                  {row.provider}
                </span>
                <span className={shared.gridCell} role="gridcell">
                  {row.last_success_at ?? '-'}
                </span>
                <span className={shared.gridCell} role="gridcell">
                  {row.dlq_pending_count ?? 0}
                </span>
              </div>
            ))
          )}
        </div>
      </section>

      <section>
        <h2 className={shared.sectionTitle}>Dead letter queue</h2>
        <div className={shared.gridTable} role="grid">
          <div className={`${shared.gridHeader} ${styles.colsDlq}`} role="row">
            <span className={shared.gridCell} role="columnheader">
              ID
            </span>
            <span className={shared.gridCell} role="columnheader">
              Campaign
            </span>
            <span className={shared.gridCell} role="columnheader">
              Event
            </span>
            <span className={shared.gridCell} role="columnheader">
              Failures
            </span>
            <span className={shared.gridCell} role="columnheader">
              Status
            </span>
            <span className={shared.gridCell} role="columnheader">
              Action
            </span>
          </div>
          {dlq.length === 0 && !loading ? (
            <EmptyState message="DLQ is empty." />
          ) : (
            dlq.map((row) => {
              const rowId = row.id ?? 0;
              return (
                <div
                  key={rowId}
                  className={`${shared.gridRow} ${styles.colsDlq}`}
                  role="row"
                  data-testid={`postback-dlq-row-${rowId}`}
                >
                  <span className={shared.gridCell} role="gridcell">
                    {rowId}
                  </span>
                  <span className={shared.gridCell} role="gridcell">
                    {row.campaign_id}
                  </span>
                  <span className={shared.gridCell} role="gridcell">
                    {row.event_type}
                  </span>
                  <span className={shared.gridCell} role="gridcell">
                    {row.failures_count}
                  </span>
                  <span className={shared.gridCell} role="gridcell">
                    {row.status}
                  </span>
                  <span className={shared.gridCell} role="gridcell">
                    {canWrite && row.status !== 'RETRIED' ? (
                      <Button
                       
                        variant="secondary"
                        disabled={retryBusyId === rowId}
                        data-testid={`postback-dlq-retry-${rowId}`}
                        data-row-id={String(rowId)}
                        onClick={onRetryDlqClick}
                      >
                        Retry
                      </Button>
                    ) : (
                      '-'
                    )}
                  </span>
                </div>
              );
            })
          )}
        </div>
      </section>
    </div>
  );
}
