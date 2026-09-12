import { campaignPanelError } from '@/domains/campaigns/editor/campaign_editor_shared';
import type { CampaignFraudSignalsWorkspace } from '@/domains/campaigns/editor/use_campaign_fraud_signals_workspace';
import { useCampaignFraudSignalsWorkspace } from '@/domains/campaigns/editor/use_campaign_fraud_signals_workspace';
import { adminSpacing } from '@/lib/admin_spacing';
import { sessionHasPermission } from '@/lib/session_permissions';
import { useSession } from '@/hooks/use_session';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  DirectoryFilterForm,
  FilterField,
  INLINE_FILTER_ACTION_GRID_CLASS,
  FilterPanel,
} from '@/shell/filter_panel';

export type CampaignFraudSignalsPanelProps = {
  workspace: CampaignFraudSignalsWorkspace;
};

function ProbeSummaryRows({ summary }: { summary: CampaignFraudSignalsWorkspace['probeSummary'] }) {
  if (!summary) {
    return null;
  }
  return (
    <dl className={`grid ${adminSpacing.gap.sm}`}>
      <div>Cluster: {summary.cluster_id ?? ''}</div>
      <div>Sessions: {summary.session_count ?? ''}</div>
      <div>Campaigns: {summary.campaign_count ?? ''}</div>
      <div>Verify count: {summary.verify_count ?? ''}</div>
      <div>Avg probe score: {summary.avg_probe_score ?? ''}</div>
      {summary.ja3 ? <div>JA3: {summary.ja3}</div> : null}
      {summary.ja4 ? <div>JA4: {summary.ja4}</div> : null}
      {summary.tcp_sig ? <div>TCP sig: {summary.tcp_sig}</div> : null}
      {summary.webgl ? <div>WebGL: {summary.webgl}</div> : null}
      <div>Export done: {summary.export_done ? 'yes' : 'no'}</div>
    </dl>
  );
}

function CrowdSummaryRows({ summary }: { summary: CampaignFraudSignalsWorkspace['crowdSummary'] }) {
  if (!summary) {
    return null;
  }
  return (
    <dl className={`grid ${adminSpacing.gap.sm}`}>
      <div>Active: {summary.active ? 'yes' : 'no'}</div>
      <div>Score: {summary.score}</div>
      <div>Unique clusters: {summary.unique_clusters}</div>
      <div>Simhash neighbors: {summary.simhash_neighbors}</div>
      <div>Entry count: {summary.entry_count}</div>
    </dl>
  );
}

export function CampaignFraudSignalsPanel({ workspace }: CampaignFraudSignalsPanelProps) {
  const {
    clusterId,
    setClusterId,
    probeLoading,
    crowdLoading,
    probeError,
    crowdError,
    probeSummary,
    crowdSummary,
    onLoadCrowdWave,
    onLoadProbeCluster,
  } = workspace;

  const { user } = useSession();
  const canRead = sessionHasPermission(user?.permissions, 'audit:read');
  if (!canRead) {
    return (
      <FilterPanel>
        <h3>Fraud signals</h3>
        <p>Requires audit:read permission.</p>
      </FilterPanel>
    );
  }

  const probeBusy = probeLoading;
  const crowdBusy = crowdLoading;

  return (
    <FilterPanel>
      <h3>Fraud signals</h3>
      <p>Campaign crowd-wave snapshot and probe-cluster lookup (audit:read).</p>

      <div className={`grid ${adminSpacing.gap.lg}`}>
        <div className={`grid ${adminSpacing.gap.md}`}>
          <h4>Crowd wave</h4>
          <Button disabled={crowdBusy} onClick={onLoadCrowdWave} type="button" variant="outline">
            {crowdBusy ? 'Loading...' : 'Load crowd wave'}
          </Button>
          {crowdError ? campaignPanelError(crowdError, 'Could not load crowd wave') : null}
          <CrowdSummaryRows summary={crowdSummary} />
        </div>

        <div className={`grid ${adminSpacing.gap.md}`}>
          <h4>Probe cluster</h4>
          <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
            <FilterField htmlFor="fraud-probe-cluster-id" label="Cluster ID (32 hex)">
              <Input
                id="fraud-probe-cluster-id"
                value={clusterId}
                onChange={(event) => setClusterId(event.target.value)}
              />
            </FilterField>
            <div className={INLINE_FILTER_ACTION_GRID_CLASS}>
              <Button
                disabled={probeBusy || clusterId.trim().length === 0}
                onClick={onLoadProbeCluster}
                type="button"
                variant="outline"
              >
                {probeBusy ? 'Loading...' : 'Load probe cluster'}
              </Button>
            </div>
          </DirectoryFilterForm>
          {probeError ? campaignPanelError(probeError, 'Could not load probe cluster') : null}
          <ProbeSummaryRows summary={probeSummary} />
        </div>
      </div>
    </FilterPanel>
  );
}

export function CampaignFraudSignalsPanelWithWorkspace({ campaignId }: { campaignId: string }) {
  const workspace = useCampaignFraudSignalsWorkspace(campaignId);
  return <CampaignFraudSignalsPanel workspace={workspace} />;
}
