import { useEffect, useState, type FormEvent } from 'react';
import { to } from '../lib/to.js';
import { apiConfirmed } from '../helpers/confirmed_api.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import { Button } from '../components/button.js';
import { Checkbox } from '../components/checkbox.js';
import { FormField } from '../components/form_field.js';

type EulaPayload = {
  version?: string;
  text?: string;
};

type BootstrapResponse = {
  click_url_template?: string;
};

export function BootstrapPage() {
  const [installToken, setInstallToken] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [trackingDomain, setTrackingDomain] = useState('');
  const [eulaText, setEulaText] = useState('');
  const [eulaVersion, setEulaVersion] = useState('');
  const [eulaAccepted, setEulaAccepted] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    void (async () => {
      const [res, err] = await to(
        fetch('/api/v1/eula', { credentials: 'same-origin' }).then(async (r) => {
          if (!r.ok) throw new Error('eula unavailable');
          return { data: (await r.json()) as EulaPayload };
        })
      );
      if (err || !res?.data) return;
      setEulaVersion(res.data.version || '');
      setEulaText(res.data.text || '');
    })();
  }, []);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    const [res, err] = await to(
      apiConfirmed('/api/v1/settings/platform/bootstrap', {
        method: 'POST',
        headers: {
          'X-Install-Token': installToken,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          config: {
            tracking_domain: trackingDomain,
            default_currency: 'USD',
            timezone: 'UTC',
            ingress_schema: 'ad_event_processor_native',
            profile: 'single_vps',
            network_interface: 'eth0',
            telemetry_enabled: true,
            edge_xdp: false,
            edge_expose_click: true,
            edge_expose_openrtb: false,
            stripe: { enabled: false },
          },
          admin_email: email,
          admin_password: password,
          eula_version: eulaAccepted ? eulaVersion : undefined,
        }),
      })
    );
    if (err) {
      if (err instanceof ConfirmCancelledError) {
        setLoading(false);
        return;
      }
      setError(mapServiceError(err).message);
      setLoading(false);
      return;
    }
    sessionStorage.setItem('install_tracking_domain', trackingDomain);
    const data = res?.data as BootstrapResponse | null | undefined;
    if (data?.click_url_template) {
      sessionStorage.setItem('install_click_url', data.click_url_template);
    }
    sessionStorage.setItem('install_ingress_enabled', '0');
    window.location.assign('/install/done');
  };

  return (
    <div className="login-page">
      <div className="login-box login-box--narrow">
        <h1 className="login-box__title">Bootstrap</h1>
        <p className="login-box__sub">Platform bootstrap</p>
        {error ? <div className="text-danger text-sm mb-3">{error}</div> : null}

        <form onSubmit={(e) => void handleSubmit(e)}>
          <FormField label="Install token" htmlFor="bootstrap-install-token">
            <input
              id="bootstrap-install-token"
              type="password"
              className="form-input"
              required
              value={installToken}
              onChange={(e) => setInstallToken(e.target.value)}
            />
          </FormField>
          <FormField label="Admin email" htmlFor="bootstrap-admin-email">
            <input
              id="bootstrap-admin-email"
              type="email"
              className="form-input"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </FormField>
          <FormField label="Admin password" htmlFor="bootstrap-admin-password">
            <input
              id="bootstrap-admin-password"
              type="password"
              className="form-input"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </FormField>
          <FormField label="Tracking domain" htmlFor="bootstrap-tracking-domain">
            <input
              id="bootstrap-tracking-domain"
              className="form-input"
              required
              value={trackingDomain}
              onChange={(e) => setTrackingDomain(e.target.value)}
            />
          </FormField>

          {eulaText ? (
            <div className="form-field">
              <pre
                className="text-sm"
                style={{ maxHeight: '160px', overflow: 'auto', whiteSpace: 'pre-wrap' }}
              >
                {eulaText}
              </pre>
              <Checkbox
                label="I accept the on-premise license agreement"
                checked={eulaAccepted}
                onChange={setEulaAccepted}
              />
            </div>
          ) : null}

          <div className="form-actions">
            <Button
              label={loading ? 'Initializing...' : 'Initialize platform'}
              variant="primary"
              type="submit"
              className="btn--block"
              loading={loading}
              disabled={loading || (Boolean(eulaText) && !eulaAccepted)}
            />
          </div>
        </form>
      </div>
    </div>
  );
}
