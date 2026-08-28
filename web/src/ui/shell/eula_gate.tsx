import { useState, type FormEvent } from 'react';
import { to } from '../../lib/to.js';
import { api } from '../../helpers/api_client.js';
import { Button } from '../system/button.js';
import { ErrorBlock } from '../system/error_block.js';
import styles from './eula_gate.module.css';

export type EulaGateProps = {
  version: string;
  text: string;
  onAccepted: () => void;
};

export function EulaGate({ version, text, onAccepted }: EulaGateProps) {
  const [checked, setChecked] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const handleSubmit = async (e: FormEvent) => {
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
      setError(err);
      return;
    }
    onAccepted();
  };

  return (
    <div className={styles.page}>
      <div className={styles.card}>
        <h1 className={styles.title}>License agreement</h1>
        <p className={styles.subtitle}>{`Version ${version}`}</p>
        <pre className={styles.eulaText}>{text || ''}</pre>
        {error ? <ErrorBlock error={error} fallbackTitle="Acceptance failed" /> : null}
        <form className={styles.form} onSubmit={(e) => void handleSubmit(e)}>
          <label className={styles.check}>
            <input
              type="checkbox"
              checked={checked}
              onChange={(e) => setChecked(e.target.checked)}
            />
            <span>I accept the ad-event-processor on-premise license agreement</span>
          </label>
          <div className={styles.actions}>
            <Button
              type="submit"
              variant="primary"
              className={styles.submit}
              disabled={loading || !checked}
            >
              {loading ? 'Saving...' : 'Continue'}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
