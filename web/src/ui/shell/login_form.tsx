import { useState, type FormEvent } from 'react';
import { to } from '../../lib/to.js';
import { api } from '../../helpers/api_client.js';
import * as auth from '../../helpers/auth.js';
import type { AuthUser } from '../../helpers/auth.js';
import { Button } from '../system/button.js';
import { ErrorBlock } from '../system/error_block.js';
import styles from './login_form.module.css';

const REASON_MESSAGES: Record<string, string> = {
  session: 'Your session expired. Sign in again.',
};

type LoginResponse = {
  user?: AuthUser;
};

export type LoginFormProps = {
  reason?: string | null;
  gateLoading?: boolean;
};

export function LoginForm({ reason = null, gateLoading = false }: LoginFormProps) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    const [res, err] = await to(
      api('/api/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email, password }),
      })
    );
    if (err) {
      setError(err);
      setLoading(false);
      return;
    }
    const csrf = res?.headers.get('X-CSRF-Token');
    if (csrf) auth.setCsrfFromLoginResponse(csrf);
    const data = res?.data as LoginResponse | null;
    if (data?.user) auth.setUser(data.user);
    window.location.assign('/');
  };

  if (gateLoading) {
    return (
      <div className={styles.page}>
        <div className={styles.card}>
          <p className={styles.loading}>Loading...</p>
        </div>
      </div>
    );
  }

  const reasonMessage = reason ? REASON_MESSAGES[reason] : null;

  return (
    <div className={styles.page}>
      <div className={styles.card}>
        <div className={styles.header}>
          <h1 className={styles.title}>ad-event-processor</h1>
          <p className={styles.subtitle}>Admin Control Plane</p>
        </div>

        {reasonMessage ? <p className={styles.noticeWarning}>{reasonMessage}</p> : null}
        {error ? <ErrorBlock error={error} fallbackTitle="Sign in failed" /> : null}

        <form className={styles.form} onSubmit={(e) => void handleSubmit(e)}>
          <div className={styles.field}>
            <label className={styles.label} htmlFor="login-email">
              Email
            </label>
            <input
              id="login-email"
              type="email"
              className={styles.input}
              required
              autoComplete="username"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>
          <div className={styles.field}>
            <label className={styles.label} htmlFor="login-password">
              Password
            </label>
            <input
              id="login-password"
              type="password"
              className={styles.input}
              required
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>
          <div className={styles.actions}>
            <Button
              type="submit"
              variant="primary"
              className={styles.submit}
              disabled={loading}
            >
              {loading ? 'Signing in...' : 'Sign in'}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
