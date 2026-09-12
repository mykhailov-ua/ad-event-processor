import { useMemo, useState } from 'react';

import { EmptyState } from '@/shell/empty_state';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
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
import type { AdminValidationError } from '@/lib/admin_validation_error';
import { adminTypography } from '@/lib/admin_kit';
import type { DirectoryOverviewField } from '@/shell/directory_overview_dialog';
import {
  DirectorySelectOverviewTable,
  directoryRecordMap,
  directoryOperateRows,
} from '@/shell/directory_select_overview_table';
import { DirectoryRowActionsMenu } from '@/shell/directory_row_actions_menu';
import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
import { TableHost } from '@/shell/ui_bands';

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
    formValidationError?: AdminValidationError;
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

function postbackConfigId(row: PostbackConfig): string {
  return `${row.campaign_id}-${row.provider}`;
}

function postbackCampaignProviderId(row: { campaign_id: string; provider: string }): string {
  return `${row.campaign_id}-${row.provider}`;
}

function postbackDlqId(row: PostbackDlqEntry): string | null {
  return row.id != null ? String(row.id) : null;
}

function buildPostbackConfigOverviewFields(row: PostbackConfig): DirectoryOverviewField[] {
  return [
    {
      label: 'Campaign',
      value: <span className={adminTypography.monoData}>{row.campaign_id}</span>,
    },
    { label: 'Provider', value: row.provider },
    { label: 'Target event', value: row.target_event },
    { label: 'URL template', value: row.url_template },
    { label: 'Test event code', value: row.test_event_code ?? '' },
    { label: 'Token', value: row.has_api_token ? 'set' : 'missing' },
  ];
}

function buildPostbackDlqOverviewFields(row: PostbackDlqEntry): DirectoryOverviewField[] {
  return [
    { label: 'ID', value: row.id ?? '' },
    {
      label: 'Campaign',
      value: <span className={adminTypography.monoData}>{row.campaign_id ?? ''}</span>,
    },
    { label: 'Click ID', value: row.click_id ?? '' },
    { label: 'Event', value: row.event_type ?? '' },
    { label: 'Status', value: row.status ?? '' },
    { label: 'Failures', value: row.failures_count ?? '' },
    { label: 'Last error', value: row.last_error ?? '' },
  ];
}

function buildPostbackStatusOverviewFields(row: PostbackCampaignStatus): DirectoryOverviewField[] {
  return [
    {
      label: 'Campaign',
      value: <span className={adminTypography.monoData}>{row.campaign_id}</span>,
    },
    { label: 'Provider', value: row.provider },
    { label: 'Last success', value: displayTimestamp(row.last_success_at) },
    {
      label: 'DLQ pending',
      value:
        row.dlq_pending_count > 0 ? (
          <Badge variant="destructive">{row.dlq_pending_count}</Badge>
        ) : (
          row.dlq_pending_count
        ),
    },
  ];
}

function buildPostbackHealthOverviewFields(row: PostbackHealthRow): DirectoryOverviewField[] {
  return [
    {
      label: 'Campaign',
      value: <span className={adminTypography.monoData}>{row.campaign_id}</span>,
    },
    { label: 'Provider', value: row.provider },
    {
      label: 'Success 24h',
      value: row.success_rate_24h != null ? `${row.success_rate_24h}%` : 'n/a',
    },
    {
      label: 'p95 latency',
      value: row.p95_latency_ms != null ? `${row.p95_latency_ms} ms` : 'n/a',
    },
    {
      label: 'Status',
      value: (
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
      ),
    },
    { label: 'Last error', value: row.last_error ?? '' },
    { label: 'DLQ pending', value: row.dlq_pending_count },
  ];
}

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
  const [selectedConfigId, setSelectedConfigId] = useState<string | null>(null);
  const [selectedDlqId, setSelectedDlqId] = useState<string | null>(null);
  const [selectedStatusId, setSelectedStatusId] = useState<string | null>(null);
  const [selectedHealthId, setSelectedHealthId] = useState<string | null>(null);

  const configRecordById = useMemo(() => directoryRecordMap(configs, postbackConfigId), [configs]);
  const configRows = useMemo(
    () =>
      directoryOperateRows(
        configs,
        postbackConfigId,
        (row) => `${row.provider} (${row.campaign_id})`
      ),
    [configs]
  );

  const dlqRecordById = useMemo(() => directoryRecordMap(dlq, postbackDlqId), [dlq]);
  const dlqRows = useMemo(
    () => directoryOperateRows(dlq, postbackDlqId, (row) => row.click_id ?? String(row.id ?? '')),
    [dlq]
  );

  const statusRecordById = useMemo(
    () => directoryRecordMap(campaignStatus, postbackCampaignProviderId),
    [campaignStatus]
  );
  const statusRows = useMemo(
    () =>
      directoryOperateRows(
        campaignStatus,
        postbackCampaignProviderId,
        (row) => `${row.provider} (${row.campaign_id})`
      ),
    [campaignStatus]
  );

  const healthRecordById = useMemo(
    () => directoryRecordMap(healthRows, postbackCampaignProviderId),
    [healthRows]
  );
  const healthTableRows = useMemo(
    () =>
      directoryOperateRows(
        healthRows,
        postbackCampaignProviderId,
        (row) => `${row.provider} (${row.campaign_id})`
      ),
    [healthRows]
  );

  const handleConfigSelectedIdChange = (id: string | null) => {
    setSelectedConfigId(id);
    if (id) {
      const record = configRecordById.get(id);
      if (record) {
        configForm.onPrefillFromConfig(record);
      }
    }
  };

  return (
    <IntegrationsPageWithLoad
      blockingErrorTitle="Could not load postbacks"
      fetchState={{ error, fetching, hasSnapshot }}
      title="Postbacks"
    >
      <div>
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
        <section>
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
            formValidationError={configForm.formValidationError}
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

          <div>
            <h2>Configs</h2>
            {configs.length === 0 ? (
              <EmptyState title="No configs" description="No postback configs are configured." />
            ) : (
              <TableHost>
                <DirectorySelectOverviewTable
                  buildOverviewFields={buildPostbackConfigOverviewFields}
                  disabled={fetching}
                  overviewTitle={(row) => `${row.provider} (${row.campaign_id})`}
                  recordById={configRecordById}
                  rows={configRows}
                  selectedId={selectedConfigId}
                  onSelectedIdChange={handleConfigSelectedIdChange}
                />
              </TableHost>
            )}
          </div>
        </section>
      ) : null}

      {tab === 'dlq' ? (
        <section>
          <h2>DLQ</h2>
          {dlq.length === 0 ? (
            <EmptyState title="DLQ empty" description="No failed postback deliveries in DLQ." />
          ) : (
            <TableHost>
              <DirectorySelectOverviewTable
                buildOverviewFields={buildPostbackDlqOverviewFields}
                disabled={fetching || dlqActions.retryingId != null}
                overviewTitle={(row) => row.click_id ?? String(row.id ?? 'DLQ entry')}
                recordById={dlqRecordById}
                renderActions={(tableRow, record, openOverview) => {
                  const rowId = postbackDlqId(record) ?? '';
                  const retrying = dlqActions.retryingId === rowId;
                  return (
                    <DirectoryRowActionsMenu
                      ariaLabel={`Actions for ${String(tableRow.label)}`}
                      disabled={!rowId || fetching || dlqActions.retryingId != null}
                      onOverview={openOverview}
                    >
                      <DropdownMenuItem
                        disabled={!rowId || retrying || dlqActions.retryingId != null}
                        onClick={() => {
                          if (rowId) {
                            dlqActions.onRetry(rowId);
                          }
                        }}
                      >
                        {retrying ? 'Retrying...' : 'Retry'}
                      </DropdownMenuItem>
                    </DirectoryRowActionsMenu>
                  );
                }}
                rows={dlqRows}
                selectedId={selectedDlqId}
                onSelectedIdChange={setSelectedDlqId}
              />
            </TableHost>
          )}
          {dlqActions.retryError
            ? integrationsPanelError(dlqActions.retryError, 'DLQ retry failed')
            : null}
        </section>
      ) : null}

      {tab === 'status' ? (
        <section>
          <h2>Campaign status</h2>
          {campaignStatus.length === 0 ? (
            <EmptyState
              title="No campaign status"
              description="No postback delivery status rows returned."
            />
          ) : (
            <TableHost>
              <DirectorySelectOverviewTable
                buildOverviewFields={buildPostbackStatusOverviewFields}
                disabled={fetching}
                overviewTitle={(row) => `${row.provider} (${row.campaign_id})`}
                recordById={statusRecordById}
                rows={statusRows}
                selectedId={selectedStatusId}
                onSelectedIdChange={setSelectedStatusId}
              />
            </TableHost>
          )}
        </section>
      ) : null}

      {tab === 'health' ? (
        <section>
          <div>
            <h2>Delivery health (24h)</h2>
            <span>Alert when success rate drops below {healthAlertThreshold}%</span>
            {healthRunbookPath ? <a href={healthRunbookPath}>Runbook</a> : null}
          </div>
          {healthFetching && !hasHealthSnapshot ? <p>Loading health metrics...</p> : null}
          {healthError ? integrationsPanelError(healthError, 'Health load failed') : null}
          {healthRows.length === 0 && hasHealthSnapshot ? (
            <EmptyState title="No health rows" description="No postback configs to aggregate." />
          ) : null}
          {healthRows.length > 0 ? (
            <TableHost>
              <DirectorySelectOverviewTable
                buildOverviewFields={buildPostbackHealthOverviewFields}
                disabled={healthFetching}
                overviewTitle={(row) => `${row.provider} (${row.campaign_id})`}
                recordById={healthRecordById}
                renderActions={(tableRow, record, openOverview) => (
                  <DirectoryRowActionsMenu
                    ariaLabel={`Actions for ${String(tableRow.label)}`}
                    disabled={healthFetching}
                    onOverview={openOverview}
                  >
                    {record.dlq_pending_count > 0 ? (
                      <DropdownMenuItem onClick={() => onTabChange('dlq')}>
                        {record.dlq_pending_count} pending in DLQ
                      </DropdownMenuItem>
                    ) : null}
                  </DirectoryRowActionsMenu>
                )}
                rows={healthTableRows}
                selectedId={selectedHealthId}
                onSelectedIdChange={setSelectedHealthId}
              />
            </TableHost>
          ) : null}
        </section>
      ) : null}
    </IntegrationsPageWithLoad>
  );
}
