import { useCallback, useEffect, useState } from 'react';
import {
  fetchFraudLabels,
  isValidFraudIPHash,
  postFraudLabel,
  type FraudManualLabel,
} from '../helpers/fraud_api.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { Button } from './button.js';

export type FraudLabelsPanelProps = {
  customerId: string;
  canWrite: boolean;
};

export function FraudLabelsPanel({ customerId, canWrite }: FraudLabelsPanelProps) {
  const [labels, setLabels] = useState<FraudManualLabel[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [ipHash, setIPHash] = useState('');
  const [reason, setReason] = useState('');

  const load = useCallback(async () => {
    if (!customerId) return;
    setLoading(true);
    setError(null);
    try {
      const rows = await fetchFraudLabels(customerId, 50);
      setLabels(rows);
    } catch (err) {
      setError(mapServiceError(err).message);
    } finally {
      setLoading(false);
    }
  }, [customerId]);

  useEffect(() => {
    void load();
  }, [load]);

  const submitLabel = async (label: number, hashOverride?: string) => {
    if (!canWrite || !customerId) return;
    const hash = (hashOverride ?? ipHash).trim().toLowerCase();
    if (!isValidFraudIPHash(hash)) {
      pushToastMessage({ title: 'Invalid IP hash', message: 'Enter 32 hex characters.' });
      return;
    }
    setSaving(true);
    setError(null);
    try {
      await postFraudLabel(customerId, { ip_hash: hash, label, reason: reason.trim() });
      pushToastMessage({
        title: 'Label saved',
        message: label === 1 ? 'Marked as fraud.' : 'Marked as legit.',
      });
      if (!hashOverride) {
        setIPHash('');
        setReason('');
      }
      await load();
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      const message = mapServiceError(err).message;
      setError(message);
      pushToastMessage({ title: 'Label failed', message });
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="stack stack--lg" data-testid="fraud-labels-panel">
      {canWrite ? (
        <form
          className="stack"
          onSubmit={(e) => {
            e.preventDefault();
            void submitLabel(1);
          }}
        >
          <div className="form-grid form-grid--2">
            <label className="form-field">
              <span className="form-field__label">IP hash (32 hex)</span>
              <input
                className="input font-mono"
                value={ipHash}
                onChange={(e) => setIPHash(e.target.value)}
                placeholder="0123456789abcdef0123456789abcdef"
                disabled={saving}
                data-testid="fraud-label-ip-hash"
              />
            </label>
            <label className="form-field">
              <span className="form-field__label">Reason (optional)</span>
              <input
                className="input"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                disabled={saving}
                data-testid="fraud-label-reason"
              />
            </label>
          </div>
          <div className="button-row">
            <Button
              label="Mark fraud"
              variant="danger"
              type="submit"
              disabled={saving}
              data-testid="fraud-label-submit-fraud"
            />
            <Button
              label="Mark legit"
              variant="secondary"
              type="button"
              disabled={saving}
              onClick={() => void submitLabel(0)}
              data-testid="fraud-label-submit-legit"
            />
          </div>
        </form>
      ) : null}

      {error ? <p className="text-danger text-sm">{error}</p> : null}

      {loading ? (
        <p className="loading-hint">Loading labels…</p>
      ) : labels.length === 0 ? (
        <p className="text-muted text-sm">No manual labels for this customer yet.</p>
      ) : (
        <div className="table-wrapper">
          <table className="data-table">
            <thead>
              <tr>
                <th scope="col">IP hash</th>
                <th scope="col">Label</th>
                <th scope="col">Reason</th>
                <th scope="col">Created</th>
                {canWrite ? <th scope="col" /> : null}
              </tr>
            </thead>
            <tbody>
              {labels.map((row) => (
                <tr key={row.ip_hash} data-testid={`fraud-label-row-${row.ip_hash.slice(0, 8)}`}>
                  <td className="font-mono text-sm">{`${row.ip_hash.slice(0, 8)}…`}</td>
                  <td>{row.label === 1 ? 'fraud' : 'legit'}</td>
                  <td>{row.reason ?? '—'}</td>
                  <td className="text-sm text-muted">{row.created_at ?? '—'}</td>
                  {canWrite ? (
                    <td>
                      <div className="button-row">
                        <Button
                          label="Fraud"
                          variant="ghost"
                          size="sm"
                          disabled={saving}
                          onClick={() => void submitLabel(1, row.ip_hash)}
                        />
                        <Button
                          label="Legit"
                          variant="ghost"
                          size="sm"
                          disabled={saving}
                          onClick={() => void submitLabel(0, row.ip_hash)}
                        />
                      </div>
                    </td>
                  ) : null}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
