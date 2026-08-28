import { useState } from 'react';
import {
  fetchFraudDecision,
  fraudDecisionTierLabel,
  postFraudOverride,
  type FraudDecision,
} from '../helpers/fraud_decision_api.js';
import { isValidFraudIPHash } from '../helpers/fraud_api.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { StatusBadge } from './status_badge.js';
import { Button } from './button.js';

export type FraudDecisionLookupProps = {
  customerId: string;
  canWrite?: boolean;
};

export function FraudDecisionLookup({ customerId, canWrite = false }: FraudDecisionLookupProps) {
  const [ipHash, setIPHash] = useState('');
  const [campaignId, setCampaignId] = useState('');
  const [hours, setHours] = useState('24');
  const [loading, setLoading] = useState(false);
  const [overriding, setOverriding] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<FraudDecision | null>(null);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!customerId) return;
    const hash = ipHash.trim().toLowerCase();
    if (!isValidFraudIPHash(hash)) {
      setError('Enter a 32-character hex IP hash.');
      setResult(null);
      return;
    }
    const parsedHours = Number.parseInt(hours, 10);
    setLoading(true);
    setError(null);
    try {
      const decision = await fetchFraudDecision(customerId, {
        ip_hash: hash,
        campaign_id: campaignId.trim() || undefined,
        hours: Number.isFinite(parsedHours) && parsedHours > 0 ? parsedHours : 24,
      });
      setResult(decision);
    } catch (err) {
      setResult(null);
      setError(mapServiceError(err).message);
    } finally {
      setLoading(false);
    }
  };

  const markFalsePositive = async () => {
    if (!canWrite || !result || !customerId) return;
    setOverriding(true);
    setError(null);
    try {
      await postFraudOverride(customerId, {
        campaign_id: result.campaign_id,
        ip_hash: result.ip_hash,
      });
      pushToastMessage({
        title: 'Override queued',
        message: 'False-positive remediation will apply on edge within about 60 seconds.',
      });
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      const message = mapServiceError(err).message;
      setError(message);
      pushToastMessage({ title: 'Override failed', message });
    } finally {
      setOverriding(false);
    }
  };

  const tierBadge = (tier: string): 'ok' | 'warning' | 'failed' | 'pending' => {
    switch (tier) {
      case 'pass':
        return 'ok';
      case 'suspect':
        return 'warning';
      case 'ivt':
      case 'block':
        return 'failed';
      default:
        return 'pending';
    }
  };

  return (
    <div className="stack stack--lg" data-testid="fraud-decision-lookup">
      <form className="stack stack--sm" onSubmit={(event) => void submit(event)}>
        <label className="field">
          <span className="field__label">IP hash (32 hex)</span>
          <input
            className="field__input font-mono"
            value={ipHash}
            onChange={(event) => setIPHash(event.target.value)}
            placeholder="a1b2c3..."
            maxLength={32}
            data-testid="fraud-decision-ip-hash"
          />
        </label>
        <label className="field">
          <span className="field__label">Campaign ID (optional)</span>
          <input
            className="field__input font-mono"
            value={campaignId}
            onChange={(event) => setCampaignId(event.target.value)}
            placeholder="UUID"
            data-testid="fraud-decision-campaign-id"
          />
        </label>
        <label className="field">
          <span className="field__label">Lookback hours (max 168)</span>
          <input
            className="field__input"
            type="number"
            min={1}
            max={168}
            value={hours}
            onChange={(event) => setHours(event.target.value)}
            data-testid="fraud-decision-hours"
          />
        </label>
        <div>
          <Button
            label={loading ? 'Looking up...' : 'Why blocked?'}
            type="submit"
            disabled={loading}
          />
        </div>
      </form>

      {error ? <p className="text-danger text-sm">{error}</p> : null}

      {result ? (
        <div className="card stack stack--sm" data-testid="fraud-decision-result">
          <div className="cluster cluster--sm">
            <StatusBadge
              status={tierBadge(result.tier)}
              label={fraudDecisionTierLabel(result.tier)}
            />
            <span className="font-mono text-sm">score {result.score}</span>
            {result.score_missing ? (
              <span className="text-warning text-xs">ML score missing</span>
            ) : null}
          </div>
          <p className="text-muted text-xs">{result.disclaimer}</p>
          <dl className="definition-list">
            <dt>Campaign</dt>
            <dd className="font-mono text-sm">{result.campaign_id}</dd>
            <dt>Window</dt>
            <dd>{result.window_start}</dd>
            <dt>Evaluated</dt>
            <dd>{result.evaluated_at}</dd>
            <dt>ML probability</dt>
            <dd className="font-mono">{result.ml_probability.toFixed(4)}</dd>
            <dt>Adjusted probability</dt>
            <dd className="font-mono">{result.adjusted_probability.toFixed(4)}</dd>
            {result.model_name ? (
              <>
                <dt>Model</dt>
                <dd className="font-mono">{result.model_name}</dd>
              </>
            ) : null}
          </dl>
          <div className="cluster cluster--sm text-sm">
            <span>Residential proxy: {result.residential_proxy ? 'yes' : 'no'}</span>
            <span>Structural fraud: {result.structural_fraud ? 'yes' : 'no'}</span>
            <span>FP guard: {result.fp_guard_applied ? 'yes' : 'no'}</span>
          </div>
          {canWrite && result.tier !== 'pass' ? (
            <div className="button-row">
              <Button
                label={overriding ? 'Submitting...' : 'Mark false positive'}
                variant="secondary"
                size="sm"
                disabled={overriding}
                onClick={() => void markFalsePositive()}
                data-testid="fraud-mark-false-positive"
              />
            </div>
          ) : null}
          <SubsectionFeatures features={result.features} thresholds={result.campaign_thresholds} />
        </div>
      ) : null}
    </div>
  );
}

type SubsectionFeaturesProps = {
  features: Record<string, number>;
  thresholds: FraudDecision['campaign_thresholds'];
};

function SubsectionFeatures({ features, thresholds }: SubsectionFeaturesProps) {
  const rows = Object.entries(features).sort(([a], [b]) => a.localeCompare(b));
  return (
    <div className="stack stack--sm">
      <p className="text-sm font-medium">Features (16-dim)</p>
      <div className="table-wrapper">
        <table className="data-table">
          <thead>
            <tr>
              <th scope="col">Feature</th>
              <th scope="col">Value</th>
            </tr>
          </thead>
          <tbody>
            {rows.map(([name, value]) => (
              <tr key={name}>
                <td className="font-mono text-sm">{name}</td>
                <td className="font-mono text-sm">{value.toFixed(4)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p className="text-muted text-xs">
        {`Campaign thresholds: pass<=${thresholds.pass_max}, suspect<=${thresholds.suspect_max}, ivt<=${thresholds.ivt_max}, block<=${thresholds.block_above}`}
      </p>
    </div>
  );
}
