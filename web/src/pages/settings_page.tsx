import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import { apiConfirmed } from '../helpers/confirmed_api.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { surfaceServiceErrorToast } from '../helpers/service_error_toast.js';
import { CustomerApiKeysSection } from '../components/customer_api_keys_section.js';
import { can } from '../helpers/permissions.js';
import * as auth from '../helpers/auth.js';
import { validateCurrency, validateTrackingDomain } from '../helpers/validators.js';
import {
  currencySelectOptions,
  displayLabel,
  INGRESS_SELECT_OPTIONS,
  PROFILE_SELECT_OPTIONS,
  timezoneSelectOptions,
} from '../helpers/display_labels.js';
import { devModeEnabled, setDevMode } from '../helpers/dev_mode.js';
import type { OpsDoctorSummary } from '../types/api/ops.js';
import { humanizeTechnicalDetail } from '../helpers/technical_labels.js';
import { reloadRoles } from '../helpers/ops_compliance_api.js';
import { useResource } from '../hooks/use_resource.js';
import { AlertBanner } from '../components/alert_banner.js';
import { Button } from '../components/button.js';
import { Checkbox } from '../components/checkbox.js';
import { ErrorBlock } from '../components/error_block.js';
import { FormField } from '../components/form_field.js';
import { Icon } from '../components/icon.js';
import { SectionCard } from '../components/section_card.js';
import { StatusBadge } from '../components/status_badge.js';

type PlatformStripeConfig = {
  enabled?: boolean;
  secret_key?: string;
  webhook_secret?: string;
};

type PlatformConfig = {
  tracking_domain?: string;
  default_currency?: string;
  timezone?: string;
  ingress_schema?: string;
  telemetry_enabled?: boolean;
  profile?: string;
  edge_xdp?: boolean;
  network_interface?: string;
  stripe?: PlatformStripeConfig;
};

type PlatformSettingsResponse = {
  config?: PlatformConfig;
  restart_required?: string[];
  bootstrap_complete?: boolean;
  click_url_template?: string;
  secrets?: {
    stripe_secret_key?: string;
    stripe_webhook_secret?: string;
  };
};

type FieldErrors = {
  tracking_domain?: string | null;
  default_currency?: string | null;
};

function SettingsSummaryItem({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="settings-summary__item">
      <div className="settings-summary__label">{label}</div>
      <div className="settings-summary__value">{children}</div>
    </div>
  );
}

/**
 * Platform settings with save, apply, and license renewal.
 */
export function SettingsPage() {
  const user = auth.getUser();
  const canWrite = can(user?.permissions ?? [], 'settings:write');
  const canCreateApiKey = can(user?.permissions ?? [], 'campaigns:write');

  const { data: view, loading, error, reload } = useResource<PlatformSettingsResponse>(
    '/api/v1/settings/platform',
  );

  const [form, setForm] = useState<PlatformConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const [applying, setApplying] = useState(false);
  const [restartRequired, setRestartRequired] = useState<string[]>([]);
  const [stripeSecretInput, setStripeSecretInput] = useState('');
  const [stripeWebhookInput, setStripeWebhookInput] = useState('');
  const [licenseTokenInput, setLicenseTokenInput] = useState('');
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [doctorSummary, setDoctorSummary] = useState<OpsDoctorSummary | null>(null);
  const [doctorLoading, setDoctorLoading] = useState(false);
  const [rolesReloading, setRolesReloading] = useState(false);
  const [devMode, setDevModeState] = useState(devModeEnabled());

  useEffect(() => {
    if (error) surfaceServiceErrorToast(error);
  }, [error]);

  const cfg = useMemo(() => form ?? view?.config ?? {}, [form, view?.config]);

  const loadDoctor = useCallback(async () => {
    setDoctorLoading(true);
    const [res] = await to(api<OpsDoctorSummary>('/api/v1/ops/doctor'));
    setDoctorSummary(res?.data ?? null);
    setDoctorLoading(false);
  }, []);

  useEffect(() => {
    if (view && !doctorSummary && !doctorLoading) {
      void loadDoctor();
    }
  }, [view, doctorSummary, doctorLoading, loadDoctor]);

  const updateField = (key: keyof PlatformConfig, value: unknown) => {
    setForm((prev) => ({ ...(prev ?? view?.config ?? {}), [key]: value }));
  };

  const edgeXdpDoctorCheck = doctorSummary?.checks?.find((c) => c.id === 'edge_xdp') ?? null;

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canWrite) return;
    const nextErrors: FieldErrors = {
      tracking_domain: validateTrackingDomain(cfg.tracking_domain ?? ''),
      default_currency: validateCurrency(cfg.default_currency ?? ''),
    };
    setFieldErrors(nextErrors);
    if (nextErrors.tracking_domain || nextErrors.default_currency) return;

    setSaving(true);
    const stripePatch: PlatformStripeConfig = { enabled: cfg.stripe?.enabled ?? false };
    const secretTrim = stripeSecretInput.trim();
    const webhookTrim = stripeWebhookInput.trim();
    if (secretTrim) stripePatch.secret_key = secretTrim;
    if (webhookTrim) stripePatch.webhook_secret = webhookTrim;

    const patch = {
      tracking_domain: cfg.tracking_domain,
      default_currency: cfg.default_currency,
      timezone: cfg.timezone,
      ingress_schema: cfg.ingress_schema,
      telemetry_enabled: cfg.telemetry_enabled,
      profile: cfg.profile,
      edge_xdp: cfg.edge_xdp,
      network_interface: cfg.network_interface,
      stripe: stripePatch,
    };

    const [savedRes, saveErr] = await to(apiConfirmed('/api/v1/settings/platform', {
      method: 'PATCH',
      body: JSON.stringify(patch),
    }));
    if (saveErr) {
      if (saveErr instanceof ConfirmCancelledError) {
        setSaving(false);
        return;
      }
      const v = mapServiceError(saveErr);
      pushToastMessage({ title: v.title, message: v.message, code: v.code });
      setSaving(false);
      return;
    }
    const saved = savedRes?.data as PlatformSettingsResponse | undefined;
    if (saved?.restart_required?.length) {
      setRestartRequired(saved.restart_required);
    }
    pushToastMessage({ title: 'Saved', message: 'Platform settings updated' });
    setForm(null);
    setStripeSecretInput('');
    setStripeWebhookInput('');
    reload();
    void loadDoctor();
    setSaving(false);
  };

  const handleLicenseApply = async () => {
    if (!canWrite) return;
    const token = licenseTokenInput.trim();
    if (!token) {
      pushToastMessage({ title: 'License', message: 'Paste the license JWT from your vendor' });
      return;
    }
    const [res, err] = await to(apiConfirmed('/api/v1/license/apply', {
      method: 'POST',
      body: JSON.stringify({ token }),
    }));
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      const v = mapServiceError(err);
      pushToastMessage({ title: v.title, message: v.message, code: v.code });
      return;
    }
    setLicenseTokenInput('');
    const until = (res?.data as { valid_until?: string } | undefined)?.valid_until ?? '';
    pushToastMessage({
      title: 'License updated',
      message: until ? `Valid until ${until}` : 'License applied',
    });
    reload();
  };

  const handleApply = async () => {
    if (!canWrite) return;
    setApplying(true);
    const [, applyErr] = await to(apiConfirmed('/api/v1/settings/platform/apply', { method: 'POST' }));
    if (applyErr) {
      if (applyErr instanceof ConfirmCancelledError) {
        setApplying(false);
        return;
      }
      const v = mapServiceError(applyErr);
      pushToastMessage({ title: v.title, message: v.message, code: v.code });
      setApplying(false);
      return;
    }
    pushToastMessage({ title: 'Applied', message: 'Configuration written to disk' });
    reload();
    void loadDoctor();
    setApplying(false);
  };

  if (loading && !view) {
    return <span className="text-muted">Loading…</span>;
  }

  if (error) {
    return <ErrorBlock error={error} />;
  }

  if (!view) return null;

  const restartBanner = restartRequired.length > 0 ? restartRequired : (view.restart_required ?? []);

  return (
    <div className="settings-layout">
      <div className="page-header">
        <div className="page-header__row">
          <div className="flex items-center gap-2">
            <Icon name="settings" size={20} className="text-muted" />
            <h1 className="page-header__title">Platform settings</h1>
          </div>
          {view.bootstrap_complete === false ? (
            <StatusBadge status="pending" label="Not initialized" />
          ) : (
            <StatusBadge status="active" label="Initialized" kind="service" />
          )}
        </div>
        <p className="settings-layout__intro">
          Configure tracking, deployment profile, edge features, and payment integration.
          {' '}
          Save updates the platform record; Apply writes configuration to disk on the host.
        </p>
      </div>

      {restartBanner.length > 0 ? (
        <AlertBanner
          variant="warning"
          message={`Service restart required: ${restartBanner.map((s) => displayLabel(s)).join(', ')}`}
        />
      ) : null}

      {!canWrite ? (
        <AlertBanner
          variant="info"
          message="Read-only access — you can view settings but cannot save or apply changes."
        />
      ) : null}

      <div className="settings-summary">
        <SettingsSummaryItem label="Deployment profile">{displayLabel(cfg.profile)}</SettingsSummaryItem>
        <SettingsSummaryItem label="Traffic format">{displayLabel(cfg.ingress_schema)}</SettingsSummaryItem>
        <SettingsSummaryItem label="Timezone">{cfg.timezone ?? '—'}</SettingsSummaryItem>
        <SettingsSummaryItem label="Click URL">
          {view.click_url_template ? (
            <code className="settings-summary__code" title={view.click_url_template}>
              {view.click_url_template}
            </code>
          ) : '—'}
        </SettingsSummaryItem>
      </div>

      <form className="settings-form" onSubmit={(e) => void handleSave(e)}>
        <div className="settings-grid">
          <SectionCard icon="globe" title="General" desc="Defaults used across campaigns and billing.">
            <FormField
              label="Tracking domain"
              error={fieldErrors.tracking_domain}
              hint="Hostname for impression and click tracking"
            >
              <input
                className="form-input"
                value={cfg.tracking_domain ?? ''}
                disabled={!canWrite}
                onChange={(e) => {
                  updateField('tracking_domain', e.target.value);
                  setFieldErrors((fe) => ({
                    ...fe,
                    tracking_domain: validateTrackingDomain(e.target.value),
                  }));
                }}
              />
            </FormField>
            <FormField
              label="Default currency"
              error={fieldErrors.default_currency}
              hint="Used for billing and reporting defaults"
            >
              <select
                className="form-input"
                value={(cfg.default_currency || 'USD').toUpperCase()}
                disabled={!canWrite}
                onChange={(e) => {
                  updateField('default_currency', e.target.value);
                  setFieldErrors((fe) => ({
                    ...fe,
                    default_currency: validateCurrency(e.target.value),
                  }));
                }}
              >
                {currencySelectOptions(cfg.default_currency).map((opt) => (
                  <option key={opt.value} value={opt.value}>{opt.label}</option>
                ))}
              </select>
            </FormField>
            <FormField label="Timezone" hint="Used for reporting and pacing">
              <select
                className="form-input"
                value={cfg.timezone || 'UTC'}
                disabled={!canWrite}
                onChange={(e) => updateField('timezone', e.target.value)}
              >
                {timezoneSelectOptions(cfg.timezone).map((opt) => (
                  <option key={opt.value} value={opt.value}>{opt.label}</option>
                ))}
              </select>
            </FormField>
          </SectionCard>

          <SectionCard
            icon="server"
            title="Deployment"
            desc="Runtime profile and network bindings for this installation."
          >
            <FormField label="Deployment profile">
              <select
                className="form-input"
                value={cfg.profile ?? 'single_vps'}
                disabled={!canWrite}
                onChange={(e) => updateField('profile', e.target.value)}
              >
                {PROFILE_SELECT_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>{opt.label}</option>
                ))}
              </select>
            </FormField>
            <FormField label="Traffic format">
              <select
                className="form-input"
                value={cfg.ingress_schema ?? 'espx_native'}
                disabled={!canWrite}
                onChange={(e) => updateField('ingress_schema', e.target.value)}
              >
                {INGRESS_SELECT_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>{opt.label}</option>
                ))}
              </select>
            </FormField>
            <FormField
              label="Network interface"
              hint="Host network interface used by edge acceleration when enabled"
            >
              <input
                className="form-input"
                value={cfg.network_interface ?? ''}
                disabled={!canWrite}
                onChange={(e) => updateField('network_interface', e.target.value)}
              />
            </FormField>
          </SectionCard>

          <SectionCard icon="toggle-left" title="Features" desc="Optional telemetry and edge acceleration.">
            <div className="settings-check-group">
              <Checkbox
                label="Telemetry enabled — export metrics to configured sinks"
                checked={cfg.telemetry_enabled ?? false}
                disabled={!canWrite}
                onChange={(checked) => updateField('telemetry_enabled', checked)}
              />
              <Checkbox
                label="Edge XDP (Enterprise) — kernel-level IP filtering via installer systemd"
                checked={cfg.edge_xdp ?? false}
                disabled={!canWrite}
                onChange={(checked) => updateField('edge_xdp', checked)}
              />
              <p className="text-muted text-sm">
                Requires Enterprise license (ebpf_xdp_edge), kernel BTF (6.1+), and installer units{' '}
                <code>ad-event-processor-edge-xdp</code> / <code>ad-event-processor-edge-bpf-sync</code>.
                Saving updates platform YAML only — use Apply on the host, then verify Doctor on Ops.
              </p>
              {cfg.edge_xdp ? (
                doctorLoading && !edgeXdpDoctorCheck ? (
                  <p className="text-muted text-sm settings-edge-xdp__status">Checking host runtime…</p>
                ) : !edgeXdpDoctorCheck || edgeXdpDoctorCheck.status === 'skip' ? (
                  <AlertBanner
                    variant="info"
                    message="Platform flag enabled — confirm host Apply and Doctor status on /ops before expecting kernel filtering."
                  />
                ) : edgeXdpDoctorCheck.status === 'pass' ? (
                  <div className="settings-edge-xdp__status" data-testid="edge-xdp-runtime-status">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">Host runtime</span>
                      <StatusBadge
                        status={edgeXdpDoctorCheck.status ?? ''}
                        kind="service"
                        label={displayLabel(edgeXdpDoctorCheck.status ?? '')}
                      />
                    </div>
                    {edgeXdpDoctorCheck.message ? (
                      <p className="text-muted text-sm">
                        {humanizeTechnicalDetail(edgeXdpDoctorCheck.message)}
                      </p>
                    ) : null}
                  </div>
                ) : (
                  <AlertBanner
                    variant={edgeXdpDoctorCheck.status === 'warn' ? 'warning' : 'error'}
                    message={
                      humanizeTechnicalDetail(edgeXdpDoctorCheck.message ?? 'Edge XDP host runtime not ready')
                      ?? 'Edge XDP host runtime not ready'
                    }
                  />
                )
              ) : null}
              <Checkbox
                label="Developer mode — show raw sysctl values and API paths in the UI"
                checked={devMode}
                onChange={(checked) => {
                  setDevMode(checked);
                  setDevModeState(checked);
                }}
              />
            </div>
          </SectionCard>

          <SectionCard
            icon="credit-card"
            title="Stripe"
            desc="Payment provider for wallet top-ups. Secrets are stored server-side."
          >
            <div className="settings-check-group">
              <Checkbox
                label="Stripe enabled"
                checked={cfg.stripe?.enabled ?? false}
                disabled={!canWrite}
                onChange={(checked) => updateField('stripe', { ...cfg.stripe, enabled: checked })}
              />
            </div>
            <FormField
              label="Secret key"
              htmlFor="settings-stripe-secret"
              hint="Leave empty to keep the current key"
            >
              <input
                id="settings-stripe-secret"
                type="password"
                className="form-input"
                placeholder="Enter Stripe secret key"
                disabled={!canWrite}
                value={stripeSecretInput}
                onChange={(e) => setStripeSecretInput(e.target.value)}
              />
            </FormField>
            <FormField
              label="Webhook secret"
              htmlFor="settings-stripe-webhook"
              hint="Leave empty to keep the current secret"
            >
              <input
                id="settings-stripe-webhook"
                type="password"
                className="form-input"
                placeholder="Enter webhook signing secret"
                disabled={!canWrite}
                value={stripeWebhookInput}
                onChange={(e) => setStripeWebhookInput(e.target.value)}
              />
            </FormField>
            {(view.secrets?.stripe_secret_key || view.secrets?.stripe_webhook_secret) ? (
              <div className="settings-secrets">
                {view.secrets?.stripe_secret_key ? (
                  <div className="settings-secrets__row">
                    <span>Stored secret key</span>
                    <span className="font-mono">{view.secrets.stripe_secret_key}</span>
                  </div>
                ) : null}
                {view.secrets?.stripe_webhook_secret ? (
                  <div className="settings-secrets__row">
                    <span>Stored webhook secret</span>
                    <span className="font-mono">{view.secrets.stripe_webhook_secret}</span>
                  </div>
                ) : null}
              </div>
            ) : null}
          </SectionCard>

          {canWrite ? (
            <SectionCard
              icon="users"
              title="RBAC roles file"
              desc="Reload role definitions from disk without restarting management."
            >
              <Button
                label={rolesReloading ? 'Reloading…' : 'Reload RBAC'}
                variant="secondary"
                size="sm"
                icon="refresh-cw"
                data-testid="roles-reload"
                loading={rolesReloading}
                disabled={rolesReloading}
                onClick={() => void (async () => {
                  setRolesReloading(true);
                  try {
                    const res = await reloadRoles();
                    pushToastMessage({
                      title: 'RBAC reloaded',
                      message: res.path ? `Loaded ${res.path}` : res.status,
                    });
                  } catch (e) {
                    if (e instanceof ConfirmCancelledError) return;
                    pushToastMessage({ title: 'Reload failed', message: mapServiceError(e).message });
                  } finally {
                    setRolesReloading(false);
                  }
                })()}
              />
            </SectionCard>
          ) : null}

          <SectionCard
            icon="shield"
            title="License renewal"
            desc="Offline on-prem: paste the monthly JWT from your vendor. No internet license check."
          >
            <FormField
              label="License JWT"
              hint="Paste the full token line from email; applies immediately without restart"
            >
              <textarea
                className="form-input code-block"
                rows={4}
                disabled={!canWrite}
                value={licenseTokenInput}
                placeholder="eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9..."
                onChange={(e) => setLicenseTokenInput(e.target.value)}
              />
            </FormField>
            {canWrite ? (
              <Button
                label="Apply license"
                variant="secondary"
                size="sm"
                icon="shield"
                onClick={() => void handleLicenseApply()}
              />
            ) : null}
          </SectionCard>
        </div>

        {canWrite ? (
          <div className="settings-actions">
            <p className="settings-actions__hint">
              Save persists to the database. Apply writes YAML to disk and may require service restart.
            </p>
            <div className="settings-actions__buttons">
              <Button
                label={applying ? 'Applying…' : 'Apply to disk'}
                variant="danger"
                icon="shield"
                loading={applying}
                disabled={applying}
                onClick={() => void handleApply()}
              />
              <Button
                label={saving ? 'Saving…' : 'Save'}
                variant="primary"
                icon="check"
                type="submit"
                loading={saving}
                disabled={saving}
              />
            </div>
          </div>
        ) : null}
      </form>

      <CustomerApiKeysSection canCreate={canCreateApiKey} />
    </div>
  );
}
