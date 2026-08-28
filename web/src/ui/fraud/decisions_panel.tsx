import type { FormEvent } from 'react';
import type { FraudDecisionDTO } from '../../helpers/fraud_api.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import { CustomerScopeBar } from '../integrations/customer_scope_bar.js';
import { FraudSubNav } from './fraud_sub_nav.js';
import styles from './fraud_shared.module.css';

export type DecisionsPanelProps = {
  customerId: string;
  ipHash: string;
  hours: string;
  campaignId: string;
  decision: FraudDecisionDTO | null;
  loading: boolean;
  error: unknown;
  lookupReady: boolean;
  onCustomerApply: (customerId: string) => void;
  onLookupDraftChange: (patch: { ipHash?: string; hours?: string; campaignId?: string }) => void;
  onLookup: () => void;
};

function formatProbability(value: number | undefined): string {
  if (value == null || !Number.isFinite(value)) return '-';
  return value.toFixed(4);
}

export function DecisionsPanel({
  customerId,
  ipHash,
  hours,
  campaignId,
  decision,
  loading,
  error,
  lookupReady,
  onCustomerApply,
  onLookupDraftChange,
  onLookup,
}: DecisionsPanelProps) {
  const onSubmit = (event: FormEvent) => {
    event.preventDefault();
    onLookup();
  };

  return (
    <div className={styles.root} data-testid="fraud-decisions-page">
      <PageChrome title="Fraud decisions" badge={decision?.tier ? <span>{decision.tier}</span> : null} />
      <FraudSubNav customerId={customerId} />
      <p className={styles.intro}>
        Explain lookup for a single IP hash. Requires 32-character hex ip_hash; returns tier, score,
        and feature snapshot from the fraud scorer path.
      </p>
      <CustomerScopeBar customerId={customerId} onApply={onCustomerApply} />
      <form className={styles.lookupForm} onSubmit={onSubmit}>
        <label className={styles.field}>
          <span className={styles.fieldLabel}>IP hash (32 hex)</span>
          <input
            className={styles.textInput}
            value={ipHash}
            onChange={(event) => onLookupDraftChange({ ipHash: event.target.value })}
            placeholder="abcdef0123456789abcdef0123456789"
            aria-label="IP hash"
            required
          />
        </label>
        <label className={styles.field}>
          <span className={styles.fieldLabel}>Hours window</span>
          <input
            className={styles.textInput}
            type="number"
            min={1}
            max={168}
            value={hours}
            onChange={(event) => onLookupDraftChange({ hours: event.target.value })}
            aria-label="Hours window"
          />
        </label>
        <label className={styles.field}>
          <span className={styles.fieldLabel}>Campaign ID (optional)</span>
          <input
            className={styles.textInput}
            value={campaignId}
            onChange={(event) => onLookupDraftChange({ campaignId: event.target.value })}
            placeholder="UUID"
            aria-label="Campaign ID"
          />
        </label>
        <div className={styles.actions}>
          <Button type="submit" size="sm" variant="secondary" disabled={!customerId || loading}>
            Explain
          </Button>
        </div>
      </form>
      <div className={styles.content}>
        {!customerId ? (
          <p className={styles.hint}>Set customer scope to run a fraud decision lookup.</p>
        ) : !lookupReady ? (
          <EmptyState message="Enter a valid 32-character ip_hash and click Explain." />
        ) : loading ? (
          <PageSkeleton rows={4} columns={3} />
        ) : error ? (
          <ErrorBlock error={error} fallbackTitle="Fraud decision lookup failed" onRetry={onLookup} />
        ) : !decision ? (
          <EmptyState message="No decision returned for this lookup." />
        ) : (
          <div className={styles.detailPanel}>
            {decision.disclaimer ? <p className={styles.helpText}>{decision.disclaimer}</p> : null}
            <div className={styles.detailGrid}>
              <div className={styles.detailItem}>
                <p className={styles.detailLabel}>Tier</p>
                <p className={styles.tierChip}>{decision.tier || '-'}</p>
              </div>
              <div className={styles.detailItem}>
                <p className={styles.detailLabel}>Score</p>
                <p className={styles.detailValue}>{decision.score_missing ? 'missing' : decision.score}</p>
              </div>
              <div className={styles.detailItem}>
                <p className={styles.detailLabel}>ML probability</p>
                <p className={styles.detailValue}>{formatProbability(decision.ml_probability)}</p>
              </div>
              <div className={styles.detailItem}>
                <p className={styles.detailLabel}>Adjusted probability</p>
                <p className={styles.detailValue}>{formatProbability(decision.adjusted_probability)}</p>
              </div>
              <div className={styles.detailItem}>
                <p className={styles.detailLabel}>Campaign</p>
                <p className={styles.detailValue}>{decision.campaign_id || '-'}</p>
              </div>
              <div className={styles.detailItem}>
                <p className={styles.detailLabel}>Evaluated at</p>
                <p className={styles.detailValue}>{decision.evaluated_at || '-'}</p>
              </div>
              <div className={styles.detailItem}>
                <p className={styles.detailLabel}>Window start</p>
                <p className={styles.detailValue}>{decision.window_start || '-'}</p>
              </div>
              <div className={styles.detailItem}>
                <p className={styles.detailLabel}>Residential proxy</p>
                <p className={styles.detailValue}>{decision.residential_proxy ? 'yes' : 'no'}</p>
              </div>
              <div className={styles.detailItem}>
                <p className={styles.detailLabel}>Structural fraud</p>
                <p className={styles.detailValue}>{decision.structural_fraud ? 'yes' : 'no'}</p>
              </div>
              <div className={styles.detailItem}>
                <p className={styles.detailLabel}>FP guard applied</p>
                <p className={styles.detailValue}>{decision.fp_guard_applied ? 'yes' : 'no'}</p>
              </div>
              {decision.model_name ? (
                <div className={styles.detailItem}>
                  <p className={styles.detailLabel}>Model</p>
                  <p className={styles.detailValue}>{decision.model_name}</p>
                </div>
              ) : null}
            </div>
            <div>
              <h3 className={styles.sectionTitle}>Campaign thresholds</h3>
              <div className={styles.detailGrid}>
                <div className={styles.detailItem}>
                  <p className={styles.detailLabel}>Pass max</p>
                  <p className={styles.detailValue}>{decision.campaign_thresholds?.pass_max ?? '-'}</p>
                </div>
                <div className={styles.detailItem}>
                  <p className={styles.detailLabel}>Suspect max</p>
                  <p className={styles.detailValue}>{decision.campaign_thresholds?.suspect_max ?? '-'}</p>
                </div>
                <div className={styles.detailItem}>
                  <p className={styles.detailLabel}>IVT max</p>
                  <p className={styles.detailValue}>{decision.campaign_thresholds?.ivt_max ?? '-'}</p>
                </div>
                <div className={styles.detailItem}>
                  <p className={styles.detailLabel}>Block above</p>
                  <p className={styles.detailValue}>{decision.campaign_thresholds?.block_above ?? '-'}</p>
                </div>
              </div>
            </div>
            <div>
              <h3 className={styles.sectionTitle}>Features</h3>
              <pre className={styles.featuresJson}>
                {JSON.stringify(decision.features ?? {}, null, 2)}
              </pre>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
