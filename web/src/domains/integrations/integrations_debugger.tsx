import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import { DirectoryFilterForm, FilterField, FilterPanel } from '@/shell/filter_panel';
import {
  IntegrationsPageWithLoad,
  integrationsPanelError,
} from '@/domains/integrations/integrations_nav';
import { ErrorBlock } from '@/shell/error_block';
import type { IntegrationsDebuggerPageWorkspace } from '@/domains/integrations/use_integrations_debugger_page_workspace';
import { displayTimestamp } from '@/lib/display';

export type IntegrationsDebuggerProps = IntegrationsDebuggerPageWorkspace;

export function IntegrationsDebugger({
  draftCampaignId,
  onDraftCampaignIdChange,
  onApplyCampaignId,
  formValidationError,
  loadingKey,
  busy,
  canRun,
  actionError,
  smokeResult,
  flowResult,
  postbackResult,
  onRunSmoke,
  onValidateFlow,
  onPostbackDryRun,
}: IntegrationsDebuggerProps) {
  return (
    <IntegrationsPageWithLoad
      alerts={actionError ? integrationsPanelError(actionError, 'Debugger action failed') : null}
      blockingErrorTitle="Integration debugger unavailable"
      fetchState={{ error: undefined, fetching: false, hasSnapshot: true }}
      title="Integration debugger"
    >
      <FilterPanel>
        <h2>Campaign scope</h2>
        <p>
          Run smoke, flow validation, and postback dry-run against a single campaign. Deep links
          from the campaign editor prefill the campaign ID query parameter.
        </p>
        {formValidationError ? (
          <ErrorBlock error={formValidationError} title="Check campaign scope" />
        ) : null}
        <DirectoryFilterForm
          layout="auto-fill"
          onSubmit={(event) => {
            event.preventDefault();
            onApplyCampaignId();
          }}
        >
          <FilterField htmlFor="integration-debugger-campaign-id" label="Campaign ID">
            <Input
              id="integration-debugger-campaign-id"
              required
              value={draftCampaignId}
              onChange={(event) => onDraftCampaignIdChange(event.target.value)}
            />
          </FilterField>
          <Button disabled={!canRun} type="submit" variant="outline">
            Apply scope
          </Button>
        </DirectoryFilterForm>
      </FilterPanel>

      <div>
        <Button disabled={busy || !canRun} onClick={onRunSmoke} type="button" variant="secondary">
          {loadingKey === 'smoke' ? 'Running smoke test...' : 'Run smoke test'}
        </Button>
        <Button
          disabled={busy || !canRun}
          onClick={onValidateFlow}
          type="button"
          variant="secondary"
        >
          {loadingKey === 'flow' ? 'Validating campaign...' : 'Validate campaign'}
        </Button>
        <Button
          disabled={busy || !canRun}
          onClick={onPostbackDryRun}
          type="button"
          variant="outline"
        >
          {loadingKey === 'postback' ? 'Testing postback...' : 'Test postback'}
        </Button>
      </div>

      {smokeResult ? (
        <FilterPanel>
          <h3>Smoke test</h3>
          <p>
            Result:{' '}
            <Badge variant={smokeResult.passed ? 'default' : 'destructive'}>
              {smokeResult.passed ? 'Passed' : 'Failed'}
            </Badge>
          </p>
          {smokeResult.failure_reason ? <p>{smokeResult.failure_reason}</p> : null}
          {smokeResult.final_host ? <p>Final host: {smokeResult.final_host}</p> : null}
          {smokeResult.checked_at ? (
            <p>Checked at: {displayTimestamp(smokeResult.checked_at)}</p>
          ) : null}
          {(smokeResult.redirect_chain?.length ?? 0) > 0 ? (
            <DirectoryTable>
              <TableHeader>
                <TableRow>
                  <DirectoryTableHead>URL</DirectoryTableHead>
                  <DirectoryTableHead>Status</DirectoryTableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {smokeResult.redirect_chain?.map((hop, index) => (
                  <TableRow key={`${hop.url ?? 'hop'}-${index}`}>
                    <TableCell>{hop.url ?? ''}</TableCell>
                    <TableCell>{hop.status_code ?? ''}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </DirectoryTable>
          ) : null}
        </FilterPanel>
      ) : null}

      {flowResult ? (
        <FilterPanel>
          <h3>Flow validation</h3>
          <p>
            Result:{' '}
            <Badge variant={flowResult.valid ? 'default' : 'destructive'}>
              {flowResult.valid ? 'Valid' : 'Invalid'}
            </Badge>
          </p>
          {flowResult.suggested_fix_action ? (
            <p>Suggested fix: {flowResult.suggested_fix_action}</p>
          ) : null}
          {(flowResult.path_errors?.length ?? 0) > 0 ? (
            <DirectoryTable>
              <TableHeader>
                <TableRow>
                  <DirectoryTableHead>Path index</DirectoryTableHead>
                  <DirectoryTableHead>Code</DirectoryTableHead>
                  <DirectoryTableHead>Message</DirectoryTableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {flowResult.path_errors?.map((row, index) => (
                  <TableRow key={`${row.path_index ?? 'path'}-${row.code ?? index}`}>
                    <TableCell>{row.path_index ?? ''}</TableCell>
                    <TableCell>{row.code ?? ''}</TableCell>
                    <TableCell>{row.message ?? ''}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </DirectoryTable>
          ) : null}
        </FilterPanel>
      ) : null}

      {postbackResult ? (
        <FilterPanel>
          <h3>Postback dry-run</h3>
          <p>
            Result:{' '}
            <Badge variant={postbackResult.ok ? 'default' : 'destructive'}>
              {postbackResult.ok ? 'OK' : 'Failed'}
            </Badge>{' '}
            ({postbackResult.provider})
          </p>
          {postbackResult.http_status != null ? (
            <p>HTTP status: {postbackResult.http_status}</p>
          ) : null}
          {postbackResult.target_event ? <p>Target event: {postbackResult.target_event}</p> : null}
          {postbackResult.error ? <p>{postbackResult.error}</p> : null}
          {postbackResult.rendered_url ? <p>{postbackResult.rendered_url}</p> : null}
          {(postbackResult.warnings?.length ?? 0) > 0 ? (
            <ul>{postbackResult.warnings?.map((warning) => <li key={warning}>{warning}</li>)}</ul>
          ) : null}
        </FilterPanel>
      ) : null}
    </IntegrationsPageWithLoad>
  );
}
