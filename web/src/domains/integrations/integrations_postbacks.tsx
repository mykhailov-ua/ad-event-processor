import { EmptyState } from '@/shell/empty_state';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import type {
  PostbackCampaignStatus,
  PostbackConfig,
  PostbackDlqEntry,
  PostbackDryRunResult,
  PostbackHealthRow,
} from '@/api/types';
import {
  IntegrationsPageWithLoad,
  integrationsPanelError,
} from '@/domains/integrations/integrations_nav';
import { PostbackConfigForm } from '@/domains/integrations/postback_config_form';
import { displayTimestamp } from '@/lib/display';

export type IntegrationsPostbacksTab = 'configs' | 'dlq' | 'status' | 'health';

const POSTBACKS_TABS: { id: IntegrationsPostbacksTab; label: string }[] = [
  { id: 'configs', label: 'Configs' },
  { id: 'dlq', label: 'DLQ' },
  { id: 'status', label: 'Campaign status' },
  { id: 'health', label: 'Health' },
];

export type IntegrationsPostbacksProps = {
  tab: IntegrationsPostbacksTab;
  onTabChange: (tab: IntegrationsPostbacksTab) => void;
  configs: PostbackConfig[];
  dlq: PostbackDlqEntry[];
  campaignStatus: PostbackCampaignStatus[];
  healthRows: PostbackHealthRow[];
  healthAlertThreshold: number;
  healthRunbookPath?: string;
  healthFetching: boolean;
  healthError: Error | undefined;
  hasHealthSnapshot: boolean;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  configForm: {
    draftCampaignId: string;
    draftProvider: string;
    draftUrlTemplate: string;
    draftTargetEvent: string;
    draftApiToken: string;
    draftTestEventCode: string;
    saving: boolean;
    testing: boolean;
    saveError: Error | undefined;
    testError: Error | undefined;
    saveSuccess: boolean;
    testResult: PostbackDryRunResult | undefined;
    onDraftCampaignIdChange: (value: string) => void;
    onDraftProviderChange: (value: string) => void;
    onDraftUrlTemplateChange: (value: string) => void;
    onDraftTargetEventChange: (value: string) => void;
    onDraftApiTokenChange: (value: string) => void;
    onDraftTestEventCodeChange: (value: string) => void;
    onSave: () => void;
    onTest: () => void;
    onPrefillFromConfig: (row: PostbackConfig) => void;
  };
  dlqActions: {
    retryingId: string | undefined;
    retryError: Error | undefined;
    onRetry: (id: string) => void;
  };
};

export function IntegrationsPostbacks({
  tab,
  onTabChange,
  configs,
  dlq,
  campaignStatus,
  healthRows,
  healthAlertThreshold,
  healthRunbookPath,
  healthFetching,
  healthError,
  hasHealthSnapshot,
  fetching,
  error,
  hasSnapshot,
  configForm,
  dlqActions,
}: IntegrationsPostbacksProps) {
  return (
    <IntegrationsPageWithLoad
      blockingErrorTitle="Could not load postbacks"
      fetchState={{ error, fetching, hasSnapshot }}
      title="Postbacks"
    >
      <div >
        {POSTBACKS_TABS.map((item) => (
          <Button
            key={item.id}
            type="button"
            variant={tab === item.id ? 'default' : 'outline'}
            onClick={() => onTabChange(item.id)}
          >
            {item.label}
          </Button>
        ))}
      </div>

      {tab === 'configs' ? (
        <section >
          <PostbackConfigForm
            draftCampaignId={configForm.draftCampaignId}
            draftProvider={configForm.draftProvider}
            draftUrlTemplate={configForm.draftUrlTemplate}
            draftTargetEvent={configForm.draftTargetEvent}
            draftApiToken={configForm.draftApiToken}
            draftTestEventCode={configForm.draftTestEventCode}
            saving={configForm.saving}
            testing={configForm.testing}
            saveError={configForm.saveError}
            testError={configForm.testError}
            saveSuccess={configForm.saveSuccess}
            testResult={configForm.testResult}
            onDraftCampaignIdChange={configForm.onDraftCampaignIdChange}
            onDraftProviderChange={configForm.onDraftProviderChange}
            onDraftUrlTemplateChange={configForm.onDraftUrlTemplateChange}
            onDraftTargetEventChange={configForm.onDraftTargetEventChange}
            onDraftApiTokenChange={configForm.onDraftApiTokenChange}
            onDraftTestEventCodeChange={configForm.onDraftTestEventCodeChange}
            onSave={configForm.onSave}
            onTest={configForm.onTest}
          />

          <div >
            <h2 >Configs</h2>
            {configs.length === 0 ? (
              <EmptyState title="No configs" description="No postback configs are configured." />
            ) : (
              <DirectoryTable horizontalScroll>
                <TableHeader>
                  <TableRow>
                    <DirectoryTableHead>Campaign</DirectoryTableHead>
                    <DirectoryTableHead>Provider</DirectoryTableHead>
                    <DirectoryTableHead>Target event</DirectoryTableHead>
                    <DirectoryTableHead>URL template</DirectoryTableHead>
                    <DirectoryTableHead>Test event code</DirectoryTableHead>
                    <DirectoryTableHead>Token</DirectoryTableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {configs.map((row) => (
                    <TableRow
                      key={`${row.campaign_id}-${row.provider}`}
                     
                      onClick={() => configForm.onPrefillFromConfig(row)}
                    >
                      <TableCell >{row.campaign_id}</TableCell>
                      <TableCell>{row.provider}</TableCell>
                      <TableCell>{row.target_event}</TableCell>
                      <TableCell
                       
                        title={row.url_template}
                      >
                        {row.url_template}
                      </TableCell>
                      <TableCell >{row.test_event_code ?? ''}</TableCell>
                      <TableCell>{row.has_api_token ? 'set' : 'missing'}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </DirectoryTable>
            )}
          </div>
        </section>
      ) : null}

      {tab === 'dlq' ? (
        <section >
          <h2 >DLQ</h2>
          {dlq.length === 0 ? (
            <EmptyState title="DLQ empty" description="No failed postback deliveries in DLQ." />
          ) : (
            <DirectoryTable horizontalScroll>
              <TableHeader>
                <TableRow>
                  <DirectoryTableHead>ID</DirectoryTableHead>
                  <DirectoryTableHead>Campaign</DirectoryTableHead>
                  <DirectoryTableHead>Click ID</DirectoryTableHead>
                  <DirectoryTableHead>Event</DirectoryTableHead>
                  <DirectoryTableHead>Status</DirectoryTableHead>
                  <DirectoryTableHead>Failures</DirectoryTableHead>
                  <DirectoryTableHead>Last error</DirectoryTableHead>
                  <DirectoryTableHead >Actions</DirectoryTableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {dlq.map((row) => {
                  const rowId = row.id != null ? String(row.id) : '';
                  return (
                    <TableRow key={rowId || row.campaign_id}>
                      <TableCell>{row.id}</TableCell>
                      <TableCell >{row.campaign_id ?? ''}</TableCell>
                      <TableCell >{row.click_id ?? ''}</TableCell>
                      <TableCell>{row.event_type ?? ''}</TableCell>
                      <TableCell>{row.status ?? ''}</TableCell>
                      <TableCell>{row.failures_count ?? ''}</TableCell>
                      <TableCell >{row.last_error ?? ''}</TableCell>
                      <TableCell>
                        <Button
                          disabled={!rowId || dlqActions.retryingId === rowId}
                          onClick={() => {
                            if (rowId) {
                              dlqActions.onRetry(rowId);
                            }
                          }}
                          type="button"
                          variant="outline"
                        >
                          {dlqActions.retryingId === rowId ? 'Retrying...' : 'Retry'}
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </DirectoryTable>
          )}
          {dlqActions.retryError
            ? integrationsPanelError(dlqActions.retryError, 'DLQ retry failed')
            : null}
        </section>
      ) : null}

      {tab === 'status' ? (
        <section >
          <h2 >Campaign status</h2>
          {campaignStatus.length === 0 ? (
            <EmptyState
              title="No campaign status"
              description="No postback delivery status rows returned."
            />
          ) : (
            <DirectoryTable horizontalScroll>
              <TableHeader>
                <TableRow>
                  <DirectoryTableHead>Campaign</DirectoryTableHead>
                  <DirectoryTableHead>Provider</DirectoryTableHead>
                  <DirectoryTableHead>Last success</DirectoryTableHead>
                  <DirectoryTableHead>DLQ pending</DirectoryTableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {campaignStatus.map((row) => (
                  <TableRow key={`${row.campaign_id}-${row.provider}`}>
                    <TableCell >{row.campaign_id}</TableCell>
                    <TableCell>{row.provider}</TableCell>
                    <TableCell>{displayTimestamp(row.last_success_at)}</TableCell>
                    <TableCell>
                      {row.dlq_pending_count > 0 ? (
                        <Badge variant="destructive">{row.dlq_pending_count}</Badge>
                      ) : (
                        row.dlq_pending_count
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </DirectoryTable>
          )}
        </section>
      ) : null}

      {tab === 'health' ? (
        <section >
          <div >
            <h2 >Delivery health (24h)</h2>
            <span >
              Alert when success rate drops below {healthAlertThreshold}%
            </span>
            {healthRunbookPath ? (
              <a
               
                href={healthRunbookPath}
              >
                Runbook
              </a>
            ) : null}
          </div>
          {healthFetching && !hasHealthSnapshot ? (
            <p >Loading health metrics...</p>
          ) : null}
          {healthError ? integrationsPanelError(healthError, 'Health load failed') : null}
          {healthRows.length === 0 && hasHealthSnapshot ? (
            <EmptyState title="No health rows" description="No postback configs to aggregate." />
          ) : null}
          {healthRows.length > 0 ? (
            <DirectoryTable horizontalScroll>
              <TableHeader>
                <TableRow>
                  <DirectoryTableHead>Campaign</DirectoryTableHead>
                  <DirectoryTableHead>Provider</DirectoryTableHead>
                  <DirectoryTableHead>Success 24h</DirectoryTableHead>
                  <DirectoryTableHead>p95 latency</DirectoryTableHead>
                  <DirectoryTableHead>Status</DirectoryTableHead>
                  <DirectoryTableHead>Last error</DirectoryTableHead>
                  <DirectoryTableHead >DLQ</DirectoryTableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {healthRows.map((row) => (
                  <TableRow key={`${row.campaign_id}-${row.provider}`}>
                    <TableCell >{row.campaign_id}</TableCell>
                    <TableCell>{row.provider}</TableCell>
                    <TableCell>
                      {row.success_rate_24h != null ? `${row.success_rate_24h}%` : 'n/a'}
                    </TableCell>
                    <TableCell>
                      {row.p95_latency_ms != null ? `${row.p95_latency_ms} ms` : 'n/a'}
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant={
                          row.health_status === 'fail'
                            ? 'destructive'
                            : row.health_status === 'warn'
                              ? 'secondary'
                              : 'outline'
                        }
                      >
                        {row.health_status}
                      </Badge>
                    </TableCell>
                    <TableCell >{row.last_error ?? ''}</TableCell>
                    <TableCell>
                      {row.dlq_pending_count > 0 ? (
                        <Button onClick={() => onTabChange('dlq')} type="button" variant="outline">
                          {row.dlq_pending_count} pending
                        </Button>
                      ) : (
                        '0'
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </DirectoryTable>
          ) : null}
        </section>
      ) : null}

    </IntegrationsPageWithLoad>
  );
}
