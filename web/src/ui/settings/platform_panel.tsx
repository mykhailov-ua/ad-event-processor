import { useEffect, useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import type { PlatformSettingsPatch, PlatformSettingsView } from '../../helpers/settings_api.js';
import * as auth from '../../helpers/auth.js';
import { can } from '../../helpers/permissions.js';
import { Button } from '../system/button.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import { SettingsHub } from './settings_hub.js';
import styles from './settings_shared.module.css';

export type PlatformPanelProps = {
  data: PlatformSettingsView | null;
  loading: boolean;
  error: unknown;
  saving: boolean;
  applying: boolean;
  onSave: (patch: PlatformSettingsPatch) => void;
  onApply: () => void;
};

function secretConfigured(masked?: string): boolean {
  return Boolean(masked && masked.length > 0);
}

export function PlatformPanel({
  data,
  loading,
  error,
  saving,
  applying,
  onSave,
  onApply,
}: PlatformPanelProps) {
  const permissions = auth.getUser()?.permissions ?? [];
  const canWrite = can(permissions, 'settings:write');

  const cfg = data?.config ?? {};
  const secrets = data?.secrets ?? {};

  const [trackingDomain, setTrackingDomain] = useState('');
  const [defaultCurrency, setDefaultCurrency] = useState('');
  const [timezone, setTimezone] = useState('');
  const [ingressSchema, setIngressSchema] = useState('');
  const [telemetryEnabled, setTelemetryEnabled] = useState(true);
  const [stripeEnabled, setStripeEnabled] = useState(false);
  const [stripeSecretKey, setStripeSecretKey] = useState('');
  const [stripeWebhookSecret, setStripeWebhookSecret] = useState('');
  const [checkoutSuccessUrl, setCheckoutSuccessUrl] = useState('');
  const [checkoutCancelUrl, setCheckoutCancelUrl] = useState('');
  const [edgeXdp, setEdgeXdp] = useState(false);
  const [edgeExposeClick, setEdgeExposeClick] = useState(true);
  const [edgeExposeOpenRtb, setEdgeExposeOpenRtb] = useState(false);
  const [networkInterface, setNetworkInterface] = useState('');

  useEffect(() => {
    if (!data) return;
    const c = data.config ?? {};
    setTrackingDomain(c.tracking_domain ?? '');
    setDefaultCurrency(c.default_currency ?? '');
    setTimezone(c.timezone ?? '');
    setIngressSchema(c.ingress_schema ?? '');
    setTelemetryEnabled(c.telemetry_enabled ?? true);
    setStripeEnabled(c.stripe?.enabled ?? false);
    setStripeSecretKey('');
    setStripeWebhookSecret('');
    setCheckoutSuccessUrl(c.stripe?.checkout_success_url ?? '');
    setCheckoutCancelUrl(c.stripe?.checkout_cancel_url ?? '');
    setEdgeXdp(c.edge_xdp ?? false);
    setEdgeExposeClick(c.edge_expose_click ?? true);
    setEdgeExposeOpenRtb(c.edge_expose_openrtb ?? false);
    setNetworkInterface(c.network_interface ?? '');
  }, [data]);

  if (error && !data) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load platform settings" />;
  }

  const onSubmit = (event: FormEvent) => {
    event.preventDefault();
    const patch: PlatformSettingsPatch = {
      tracking_domain: trackingDomain,
      default_currency: defaultCurrency,
      timezone,
      ingress_schema: ingressSchema,
      telemetry_enabled: telemetryEnabled,
      edge_xdp: edgeXdp,
      edge_expose_click: edgeExposeClick,
      edge_expose_openrtb: edgeExposeOpenRtb,
      network_interface: networkInterface,
      stripe: {
        enabled: stripeEnabled,
        checkout_success_url: checkoutSuccessUrl,
        checkout_cancel_url: checkoutCancelUrl,
      },
    };
    if (stripeSecretKey.trim()) {
      patch.stripe = { ...patch.stripe, secret_key: stripeSecretKey.trim() };
    }
    if (stripeWebhookSecret.trim()) {
      patch.stripe = { ...patch.stripe, webhook_secret: stripeWebhookSecret.trim() };
    }
    onSave(patch);
  };

  return (
    <div className={styles.root} data-testid="settings-platform-page">
      <PageChrome
        title="Platform settings"
        badge={
          data?.bootstrap_complete === false ? (
            <span>Bootstrap incomplete</span>
          ) : loading ? (
            <span>Loading</span>
          ) : null
        }
      />
      <SettingsHub title="More settings" showIntro={false} />

      {(data?.restart_required?.length ?? 0) > 0 ? (
        <p className={styles.restartBanner}>
          Restart required: {(data?.restart_required ?? []).join(', ')}
        </p>
      ) : null}

      {data?.click_url_template ? (
        <p className={styles.hint}>
          Click URL template: <code>{data.click_url_template}</code>
        </p>
      ) : null}

      {loading && !data ? (
        <PageSkeleton rows={6} />
      ) : (
        <form className={styles.formStack} onSubmit={onSubmit}>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>Tracking domain</span>
            <input
              className={styles.textInput}
              value={trackingDomain}
              onChange={(e) => setTrackingDomain(e.target.value)}
              disabled={!canWrite}
            />
          </label>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>Default currency</span>
            <input
              className={styles.textInput}
              value={defaultCurrency}
              onChange={(e) => setDefaultCurrency(e.target.value)}
              disabled={!canWrite}
            />
          </label>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>Timezone</span>
            <input
              className={styles.textInput}
              value={timezone}
              onChange={(e) => setTimezone(e.target.value)}
              disabled={!canWrite}
            />
          </label>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>Ingress schema</span>
            <select
              className={styles.select}
              value={ingressSchema}
              onChange={(e) => setIngressSchema(e.target.value)}
              disabled={!canWrite}
            >
              <option value="ad_event_processor_native">ad_event_processor_native</option>
              <option value="openrtb_3">openrtb_3</option>
            </select>
          </label>
          <label className={styles.checkboxRow}>
            <input
              type="checkbox"
              checked={telemetryEnabled}
              onChange={(e) => setTelemetryEnabled(e.target.checked)}
              disabled={!canWrite}
            />
            <span>Telemetry enabled</span>
          </label>
          <label className={styles.checkboxRow}>
            <input
              type="checkbox"
              checked={edgeXdp}
              onChange={(e) => setEdgeXdp(e.target.checked)}
              disabled={!canWrite}
            />
            <span>Edge XDP</span>
          </label>
          <label className={styles.checkboxRow}>
            <input
              type="checkbox"
              checked={edgeExposeClick}
              onChange={(e) => setEdgeExposeClick(e.target.checked)}
              disabled={!canWrite}
            />
            <span>Expose click on edge</span>
          </label>
          <label className={styles.checkboxRow}>
            <input
              type="checkbox"
              checked={edgeExposeOpenRtb}
              onChange={(e) => setEdgeExposeOpenRtb(e.target.checked)}
              disabled={!canWrite}
            />
            <span>Expose OpenRTB on edge</span>
          </label>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>Network interface</span>
            <input
              className={styles.textInput}
              value={networkInterface}
              onChange={(e) => setNetworkInterface(e.target.value)}
              disabled={!canWrite}
            />
          </label>

          <h3 className={styles.sectionTitle}>Stripe</h3>
          <label className={styles.checkboxRow}>
            <input
              type="checkbox"
              checked={stripeEnabled}
              onChange={(e) => setStripeEnabled(e.target.checked)}
              disabled={!canWrite}
            />
            <span>Stripe enabled</span>
          </label>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>Secret key</span>
            <input
              className={styles.textInput}
              type="password"
              autoComplete="new-password"
              value={stripeSecretKey}
              onChange={(e) => setStripeSecretKey(e.target.value)}
              disabled={!canWrite}
              placeholder={secretConfigured(secrets.stripe_secret_key) ? 'Configured (enter to replace)' : 'Not set'}
            />
            {secretConfigured(secrets.stripe_secret_key) ? (
              <span className={styles.secretHint}>Stored key ends with masked suffix only.</span>
            ) : null}
          </label>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>Webhook secret</span>
            <input
              className={styles.textInput}
              type="password"
              autoComplete="new-password"
              value={stripeWebhookSecret}
              onChange={(e) => setStripeWebhookSecret(e.target.value)}
              disabled={!canWrite}
              placeholder={
                secretConfigured(secrets.stripe_webhook_secret)
                  ? 'Configured (enter to replace)'
                  : 'Not set'
              }
            />
          </label>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>Checkout success URL</span>
            <input
              className={styles.textInput}
              value={checkoutSuccessUrl}
              onChange={(e) => setCheckoutSuccessUrl(e.target.value)}
              disabled={!canWrite}
            />
          </label>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>Checkout cancel URL</span>
            <input
              className={styles.textInput}
              value={checkoutCancelUrl}
              onChange={(e) => setCheckoutCancelUrl(e.target.value)}
              disabled={!canWrite}
            />
          </label>

          <div className={styles.toolbar}>
            {canWrite ? (
              <>
                <Button type="submit" variant="primary" disabled={saving}>
                  Save
                </Button>
                <Button type="button" disabled={applying} onClick={onApply}>
                  Apply to disk
                </Button>
              </>
            ) : (
              <p className={styles.hint}>Read-only session (settings:write required to edit).</p>
            )}
            <Link to="/ops" className={styles.bannerLink}>
              Ops doctor
            </Link>
          </div>
        </form>
      )}
    </div>
  );
}
