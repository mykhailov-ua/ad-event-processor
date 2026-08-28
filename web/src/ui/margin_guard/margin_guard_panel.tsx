import { useState } from 'react';
import { Link } from 'react-router-dom';
import * as auth from '../../helpers/auth.js';
import type { MarginGuardActivity, MarginGuardPolicy } from '../../helpers/integrations_api.js';
import { can, canReadCampaigns } from '../../helpers/permissions.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { CampaignScopeBar } from '../integrations/customer_scope_bar.js';
import shared from '../integrations/integrations_shared.module.css';
import styles from './margin_guard.module.css';

export type MarginGuardPanelProps = {
  campaignId: string;
  policies: MarginGuardPolicy[];
  activity: MarginGuardActivity[];
  loading: boolean;
  error: unknown;
  busy: boolean;
  onCampaignApply: (campaignId: string) => void;
  onCreatePolicy: (body: MarginGuardPolicy) => void;
  onRemoveOverride: (placementId: string) => void;
};

export function MarginGuardPanel({
  campaignId,
  policies,
  activity,
  loading,
  error,
  busy,
  onCampaignApply,
  onCreatePolicy,
  onRemoveOverride,
}: MarginGuardPanelProps) {
  const permissions = auth.getUser()?.permissions ?? [];
  const canWrite = can(permissions, 'campaigns:write');
  const canList = canReadCampaigns(permissions);

  const [policyName, setPolicyName] = useState('');
  const [roiFloor, setRoiFloor] = useState('0');
  const [overridePlacement, setOverridePlacement] = useState('');

  if (!canList) {
    return <ErrorBlock error={new Error('forbidden')} fallbackTitle="Margin guard access denied" />;
  }

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load margin guard" />;
  }

  return (
    <div className={shared.panelRoot} data-testid="integrations-margin-guard-page">
      <PageChrome
        title="Margin guard"
        badge={
          <Link to="/integrations" className={shared.bannerLink}>
            All integrations
          </Link>
        }
      />
      <CampaignScopeBar campaignId={campaignId} onApply={onCampaignApply} />

      {!campaignId ? (
        <div className={shared.hint}>Enter a campaign ID and apply to load policies.</div>
      ) : (
        <>
          <section>
            <h2 className={shared.sectionTitle}>Policies</h2>
            <div className={shared.gridTable} role="grid">
              <div className={`${shared.gridHeader} ${styles.colsPolicies}`} role="row">
                <span className={shared.gridCell} role="columnheader">
                  Name
                </span>
                <span className={shared.gridCell} role="columnheader">
                  Min clicks
                </span>
                <span className={shared.gridCell} role="columnheader">
                  ROI floor %
                </span>
                <span className={shared.gridCell} role="columnheader">
                  Cost/rev bps
                </span>
                <span className={shared.gridCell} role="columnheader">
                  Active
                </span>
              </div>
              {policies.length === 0 && !loading ? (
                <EmptyState message="No margin guard policies for this campaign." />
              ) : (
                policies.map((row) => (
                  <div
                    key={row.id ?? row.name}
                    className={`${shared.gridRow} ${styles.colsPolicies}`}
                    role="row"
                  >
                    <span className={shared.gridCell} role="gridcell">
                      {row.name}
                    </span>
                    <span className={shared.gridCell} role="gridcell">
                      {row.min_clicks}
                    </span>
                    <span className={shared.gridCell} role="gridcell">
                      {row.roi_floor_pct}
                    </span>
                    <span className={shared.gridCell} role="gridcell">
                      {row.cost_over_revenue_threshold_bps}
                    </span>
                    <span className={shared.gridCell} role="gridcell">
                      {row.is_active ? 'yes' : 'no'}
                    </span>
                  </div>
                ))
              )}
            </div>
            {canWrite ? (
              <div className={shared.formStack}>
                <label className={shared.field}>
                  <span className={shared.fieldLabel}>Policy name</span>
                  <input
                    className={shared.textInput}
                    value={policyName}
                    onChange={(event) => setPolicyName(event.target.value)}
                  />
                </label>
                <label className={shared.field}>
                  <span className={shared.fieldLabel}>ROI floor %</span>
                  <input
                    className={shared.textInput}
                    type="number"
                    value={roiFloor}
                    onChange={(event) => setRoiFloor(event.target.value)}
                  />
                </label>
                <Button
                 
                  variant="primary"
                  disabled={busy || !policyName.trim()}
                  onClick={() =>
                    onCreatePolicy({
                      campaign_id: campaignId,
                      name: policyName.trim(),
                      min_clicks: 0,
                      roi_floor_pct: Number.parseFloat(roiFloor) || 0,
                      zero_conv_streak: 0,
                      cost_over_revenue_threshold_bps: 0,
                      is_active: true,
                    })
                  }
                >
                  Create policy
                </Button>
              </div>
            ) : null}
          </section>

          <section>
            <h2 className={shared.sectionTitle}>Activity</h2>
            <div className={shared.gridTable} role="grid">
              <div className={`${shared.gridHeader} ${styles.colsActivity}`} role="row">
                <span className={shared.gridCell} role="columnheader">
                  Time
                </span>
                <span className={shared.gridCell} role="columnheader">
                  Placement
                </span>
                <span className={shared.gridCell} role="columnheader">
                  Action
                </span>
                <span className={shared.gridCell} role="columnheader">
                  Reason
                </span>
                <span className={shared.gridCell} role="columnheader">
                  Policy
                </span>
              </div>
              {activity.length === 0 && !loading ? (
                <EmptyState message="No margin guard activity logged." />
              ) : (
                activity.map((row) => (
                  <div
                    key={row.id ?? `${row.placement_id}-${row.created_at}`}
                    className={`${shared.gridRow} ${styles.colsActivity}`}
                    role="row"
                  >
                    <span className={shared.gridCell} role="gridcell">
                      {row.created_at}
                    </span>
                    <span className={shared.gridCell} role="gridcell">
                      {row.placement_id}
                    </span>
                    <span className={shared.gridCell} role="gridcell">
                      {row.action}
                    </span>
                    <span className={shared.gridCell} role="gridcell">
                      {row.reason}
                    </span>
                    <span className={shared.gridCell} role="gridcell">
                      {row.policy_id}
                    </span>
                  </div>
                ))
              )}
            </div>
          </section>

          {canWrite ? (
            <section>
              <h2 className={shared.sectionTitle}>Remove placement override</h2>
              <div className={shared.formStack}>
                <label className={shared.field}>
                  <span className={shared.fieldLabel}>Placement ID</span>
                  <input
                    className={shared.textInput}
                    value={overridePlacement}
                    onChange={(event) => setOverridePlacement(event.target.value)}
                  />
                </label>
                <Button
                 
                  variant="secondary"
                  disabled={busy || !overridePlacement.trim()}
                  onClick={() => onRemoveOverride(overridePlacement.trim())}
                >
                  Remove override
                </Button>
              </div>
            </section>
          ) : null}
        </>
      )}
    </div>
  );
}
