import { useEffect, useState } from 'react';
import { to } from '../lib/to.js';
import { Button } from '../components/button.js';

type HealthStatus = 'pending' | 'pass' | 'fail';

type MetaPayload = {
  bootstrap_complete?: boolean;
};

function CheckRow({ status, label }: { status: HealthStatus; label: string }) {
  const mark = status === 'pass' ? '✓' : status === 'fail' ? '✗' : '…';
  return (
    <li className={`install-check install-check--${status}`}>
      {mark} {label}
    </li>
  );
}

export function InstallDonePage() {
  const [controlHealth, setControlHealth] = useState<HealthStatus>('pending');
  const [bootstrapComplete, setBootstrapComplete] = useState(false);
  const [clickTemplate] = useState(() => sessionStorage.getItem('install_click_url') || '');
  const [trackingDomain] = useState(() => sessionStorage.getItem('install_tracking_domain') || '');
  const [ingressEnabled, setIngressEnabled] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    void (async () => {
      const [, healthErr] = await to(fetch('/health', { credentials: 'same-origin' }));
      setControlHealth(healthErr ? 'fail' : 'pass');

      const [metaRes, metaErr] = await to(
        fetch('/api/v1/meta', { credentials: 'same-origin' }).then(async (res) => {
          if (!res.ok) throw new Error('meta unavailable');
          return { data: (await res.json()) as MetaPayload };
        })
      );
      if (metaErr) {
        setError('Could not verify platform status.');
      } else if (metaRes?.data) {
        setBootstrapComplete(metaRes.data.bootstrap_complete === true);
      }

      setIngressEnabled(
        new URLSearchParams(window.location.search).get('ingress') === '1' ||
          sessionStorage.getItem('install_ingress_enabled') === '1'
      );
    })();
  }, []);

  const clickExample = clickTemplate
    ? clickTemplate.replace('{campaign_id}', 'demo').replace('{sub1}', 'test')
    : trackingDomain
      ? `https://${trackingDomain}/click?campaign_id=demo&sub1=test`
      : '';

  return (
    <div className="login-page">
      <div className="login-box login-box--narrow">
        <h1 className="login-box__title">Install complete</h1>
        <p className="login-box__sub">Verify these items before sending traffic.</p>
        {error ? <div className="text-danger text-sm mb-3">{error}</div> : null}

        <ul className="install-checklist">
          <CheckRow
            status={bootstrapComplete ? 'pass' : controlHealth === 'fail' ? 'fail' : 'pending'}
            label="Platform bootstrap saved"
          />
          <CheckRow
            status={
              controlHealth === 'pass' ? 'pass' : controlHealth === 'fail' ? 'fail' : 'pending'
            }
            label="Control API healthy"
          />
          <CheckRow
            status={trackingDomain ? 'pass' : 'pending'}
            label={
              trackingDomain
                ? `DNS: point ${trackingDomain} A-record to this server`
                : 'DNS: set TRACKING_DOMAIN in settings'
            }
          />
          <CheckRow
            status={ingressEnabled ? 'pass' : 'pending'}
            label={
              ingressEnabled
                ? 'Ingress (Caddy) enabled — HTTPS on tracking/admin hosts'
                : 'Ingress optional — enable INGRESS_ENABLED=1 for automatic TLS'
            }
          />
        </ul>

        {clickExample ? (
          <p className="text-sm mt-3">
            Sample click:{' '}
            <a href={clickExample} className="link">
              {clickExample}
            </a>
          </p>
        ) : null}

        <div className="form-actions mt-4">
          <Button
            label="Continue to login"
            variant="primary"
            className="btn--block"
            onClick={() => {
              window.location.assign('/login');
            }}
          />
        </div>

        <p className="text-muted text-sm mt-3">
          Run on server: bash scripts/install/bidshard-install.sh doctor
        </p>
      </div>
    </div>
  );
}
