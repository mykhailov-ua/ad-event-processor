import { PageChrome } from '@/components/system/page_chrome';
import { EmptyState } from '@/components/system/empty_state';
import { ErrorBlock } from '@/components/system/error_block';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type {
  PostbackCampaignStatus,
  PostbackConfig,
  PostbackDlqEntry,
  PostbackDryRunResult,
} from '@/api/types';
import { IntegrationsNav, integrationsPanelError } from '@/domains/integrations/integrations_nav';
import { PostbackConfigForm } from '@/domains/integrations/postback_config_form';
import { displayTimestamp } from '@/lib/display';

export type IntegrationsPostbacksTab = 'configs' | 'dlq' | 'status';

const POSTBACKS_TABS: { id: IntegrationsPostbacksTab; label: string }[] = [
  { id: 'configs', label: 'Configs' },
  { id: 'dlq', label: 'DLQ' },
  { id: 'status', label: 'Campaign status' },
];

export type IntegrationsPostbacksProps = {
  tab: IntegrationsPostbacksTab;
  onTabChange: (tab: IntegrationsPostbacksTab) => void;
  configs: PostbackConfig[];
  dlq: PostbackDlqEntry[];
  campaignStatus: PostbackCampaignStatus[];
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
  fetching,
  error,
  hasSnapshot,
  configForm,
  dlqActions,
}: IntegrationsPostbacksProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Postbacks">
        <IntegrationsNav />
        {integrationsPanelError(error, 'Could not load postbacks')}
      </PageChrome>
    );
  }

  return (
    <PageChrome title="Postbacks">
      <IntegrationsNav />

      <div className="flex flex-wrap gap-2">
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
      <section className="grid gap-4">
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

        <div className="grid gap-2">
        <h2 className="text-base font-semibold">Configs</h2>
        {configs.length === 0 ? (
          <EmptyState title="No configs" description="No postback configs are configured." />
        ) : (
          <div className="ui-table-frame">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Campaign</TableHead>
                  <TableHead>Provider</TableHead>
                  <TableHead>Target event</TableHead>
                  <TableHead>Token</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {configs.map((row) => (
                  <TableRow
                    key={`${row.campaign_id}-${row.provider}`}
                    className="cursor-pointer"
                    onClick={() => configForm.onPrefillFromConfig(row)}
                  >
                    <TableCell className="font-mono text-xs">{row.campaign_id}</TableCell>
                    <TableCell>{row.provider}</TableCell>
                    <TableCell>{row.target_event}</TableCell>
                    <TableCell>{row.has_api_token ? 'set' : 'missing'}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
        </div>
      </section>
      ) : null}

      {tab === 'dlq' ? (
      <section className="grid gap-2">
        <h2 className="text-base font-semibold">DLQ</h2>
        {dlq.length === 0 ? (
          <EmptyState title="DLQ empty" description="No failed postback deliveries in DLQ." />
        ) : (
          <div className="ui-table-frame">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>Campaign</TableHead>
                  <TableHead>Event</TableHead>
                  <TableHead>Failures</TableHead>
                  <TableHead>Last error</TableHead>
                  <TableHead className="w-28">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {dlq.map((row) => {
                  const rowId = row.id != null ? String(row.id) : '';
                  return (
                  <TableRow key={rowId || row.campaign_id}>
                    <TableCell>{row.id}</TableCell>
                    <TableCell className="font-mono text-xs">{row.campaign_id ?? ''}</TableCell>
                    <TableCell>{row.event_type ?? ''}</TableCell>
                    <TableCell>{row.failures_count ?? ''}</TableCell>
                    <TableCell className="max-w-md truncate">{row.last_error ?? ''}</TableCell>
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
            </Table>
          </div>
        )}
        {dlqActions.retryError ? (
          <ErrorBlock title="DLQ retry failed" message={dlqActions.retryError.message} />
        ) : null}
      </section>
      ) : null}

      {tab === 'status' ? (
      <section className="grid gap-2">
        <h2 className="text-base font-semibold">Campaign status</h2>
        {campaignStatus.length === 0 ? (
          <EmptyState
            title="No campaign status"
            description="No postback delivery status rows returned."
          />
        ) : (
          <div className="ui-table-frame">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Campaign</TableHead>
                  <TableHead>Provider</TableHead>
                  <TableHead>Last success</TableHead>
                  <TableHead>DLQ pending</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {campaignStatus.map((row) => (
                  <TableRow key={`${row.campaign_id}-${row.provider}`}>
                    <TableCell className="font-mono text-xs">{row.campaign_id}</TableCell>
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
            </Table>
          </div>
        )}
      </section>
      ) : null}

      {error && hasSnapshot ? integrationsPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
