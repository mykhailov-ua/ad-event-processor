import { useState } from 'react';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import { Button } from './button.js';

export type EulaGateProps = {
  version: string;
  text: string;
  onAccepted: () => void;
};

export function EulaGate({ version, text, onAccepted }: EulaGateProps) {
  const [checked, setChecked] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!checked) return;
    setLoading(true);
    setError(null);
    const [, err] = await to(
      api('/api/v1/eula/accept', {
        method: 'POST',
        body: JSON.stringify({ version }),
      })
    );
    setLoading(false);
    if (err) {
      setError(err.message || 'Failed to record acceptance');
      return;
    }
    onAccepted();
  };

  return (
    <div className="login-page">
      <div className="login-box login-box--narrow">
        <h1 className="login-box__title">License agreement</h1>
        <p className="login-box__sub">{`Version ${version}`}</p>
        <pre
          className="login-box__eula text-sm"
          style={{ maxHeight: '240px', overflow: 'auto', whiteSpace: 'pre-wrap' }}
        >
          {text || ''}
        </pre>
        {error ? <div className="text-danger text-sm mb-3">{error}</div> : null}
        <form onSubmit={(e) => void handleSubmit(e)}>
          <label className="form-check mb-3">
            <input
              type="checkbox"
              checked={checked}
              onChange={(e) => setChecked(e.target.checked)}
            />
            {' I accept the ad-event-processor on-premise license agreement'}
          </label>
          <div className="form-actions">
            <Button
              label={loading ? 'Saving…' : 'Continue'}
              variant="primary"
              type="submit"
              className="btn--block"
              loading={loading}
              disabled={loading || !checked}
            />
          </div>
        </form>
      </div>
    </div>
  );
}
