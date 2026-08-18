import { useEffect, useState, type FormEvent } from 'react';
import { useSearchParams } from 'react-router-dom';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import * as auth from '../helpers/auth.js';
import type { AuthUser } from '../helpers/auth.js';
import { Button } from '../components/button.js';
import { FormField } from '../components/form_field.js';

const REASON_MESSAGES: Record<string, string> = {
  session: 'Your session expired. Sign in again.',
};

type LoginResponse = {
  user?: AuthUser;
};

export function LoginPage() {
  const [searchParams] = useSearchParams();
  const reason = searchParams.get('reason');

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [gateLoading, setGateLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      const [metaRes, metaErr] = await to(api<{ bootstrap_complete?: boolean }>('/api/v1/meta'));
      if (cancelled) return;
      if (!metaErr && metaRes?.data?.bootstrap_complete === false) {
        window.location.assign('/bootstrap');
        return;
      }
      setGateLoading(false);
    })();
    return () => {
      cancelled = true;
    };
  }, []);

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
      setError(err.message || 'Login failed');
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
      <div className="login-page">
        <div className="login-box">
          <span className="text-muted">Loading…</span>
        </div>
      </div>
    );
  }

  const reasonMessage = reason ? REASON_MESSAGES[reason] : null;

  return (
    <div className="login-page">
      <div className="login-box">
        <h1 className="login-box__title">
          <span className="login-box__title-bid">Bid</span>
          <span className="login-box__title-shard">Shard</span>
        </h1>
        <p className="login-box__sub">Admin Control Plane</p>

        {reasonMessage || error ? (
          <div className="login-box__notices">
            {reasonMessage ? (
              <div className="login-box__notice login-box__notice--warning">{reasonMessage}</div>
            ) : null}
            {error ? (
              <div className="login-box__notice login-box__notice--error">{error}</div>
            ) : null}
          </div>
        ) : null}

        <form onSubmit={(e) => void handleSubmit(e)}>
          <FormField label="Email">
            <input
              type="email"
              className="form-input"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </FormField>
          <FormField label="Password">
            <input
              type="password"
              className="form-input"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </FormField>
          <div className="form-actions">
            <Button
              label={loading ? 'Signing in...' : 'Sign in'}
              variant="primary"
              type="submit"
              className="btn--block"
              loading={loading}
              disabled={loading}
            />
          </div>
        </form>
      </div>
    </div>
  );
}
