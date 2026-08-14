import type { ViewHandle } from '../lib/router_types.js';
import { el, replaceChildren, eventTargetValue } from '../lib/dom.js';
import { createResource, type ResourceState } from '../lib/fetch_resource.js';
import { to } from '../lib/to.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { apiConfirmed } from '../helpers/confirmed_api.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { mountCustomerApiKeysPanel } from './customer_api_keys_panel.js';
import { can } from '../helpers/permissions.js';
import * as auth from '../helpers/auth.js';
import { surfaceServiceErrorToast } from '../helpers/service_error_toast.js';
import { renderSelect } from '../ui/select.js';
import { renderCheckbox } from '../ui/checkbox.js';
import { renderFormField } from '../ui/form_field.js';
import { renderAlertBanner } from '../ui/alert_banner.js';
import { renderStatusBadge } from '../ui/status_badge.js';
import { validateCurrency, validateTrackingDomain } from '../helpers/validators.js';
import { renderSettingsSummaryItem } from '../ui/settings_section.js';
import { renderSectionCard } from '../ui/section_card.js';
import { renderButton } from '../ui/button.js';
import { renderIcon } from '../ui/icon.js';
import {
  currencySelectOptions,
  displayLabel,
  INGRESS_SELECT_OPTIONS,
  PROFILE_SELECT_OPTIONS,
  timezoneSelectOptions,
} from '../helpers/display_labels.js';
import { devModeEnabled, setDevMode } from '../helpers/dev_mode.js';
import { api } from '../helpers/api_client.js';
import type { OpsDoctorSummary } from '../types/api/ops.js';
import { humanizeTechnicalDetail } from '../helpers/technical_labels.js';

type PlatformStripeConfig = {
  enabled?: boolean;
  secret_key?: string;
  webhook_secret?: string;
  [key: string]: unknown;
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
  [key: string]: unknown;
};

type PlatformSettingsResponse = {
  config?: PlatformConfig;
  restart_required?: string[];
  bootstrap_complete?: boolean;
  click_url_template?: string;
  secrets?: {
    stripe_secret_key?: string;
    stripe_webhook_secret?: string;
    [key: string]: unknown;
  };
  [key: string]: unknown;
};

type FieldErrors = {
  tracking_domain?: string | null;
  default_currency?: string | null;
  [key: string]: unknown;
};

/**
 * Mount the platform settings view with save and apply actions.
 *
 * @param {HTMLElement} container
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container: HTMLElement): ViewHandle {
  let destroyed = false;
  let form: PlatformConfig | null = null;
  let saving = false;
  let applying = false;
  let restartRequired: string[] = [];
  let lastError: unknown = null;
  let stripeSecretInput = '';
  let stripeWebhookInput = '';
  let licenseTokenInput = '';
  let doctorSummary: OpsDoctorSummary | null = null;
  let doctorLoading = false;

  const user = auth.getUser();
  const canWrite = can(user?.permissions ?? [], 'settings:write');
  const canCreateApiKey = can(user?.permissions ?? [], 'campaigns:write');
  const apiKeysSlot = el('div');
  /** @type {{ destroy: () => void }|null} */
  let apiKeysPanel: { destroy?: () => void } | null = null;

  const state: ResourceState<PlatformSettingsResponse> = { data: null, loading: true, error: null };
  let fieldErrors: FieldErrors = {};

  function cfg(): PlatformConfig {
    return form ?? state.data?.config ?? {};
  }

  function updateField(key: string, value: unknown) {
    form = { ...cfg(), [key]: value };
    render();
  }

  async function loadDoctor() {
    if (doctorLoading) return;
    doctorLoading = true;
    const [res] = await to(api<OpsDoctorSummary>('/api/v1/ops/doctor'));
    doctorSummary = res?.data ?? null;
    doctorLoading = false;
    render();
  }

  function edgeXdpDoctorCheck() {
    return doctorSummary?.checks?.find((c) => c.id === 'edge_xdp') ?? null;
  }

  function renderEdgeXdpRuntimeStatus(): HTMLElement | null {
    const check = edgeXdpDoctorCheck();
    if (!cfg().edge_xdp) return null;
    if (doctorLoading && !check) {
      return el('p', { className: 'text-muted text-sm settings-edge-xdp__status' }, 'Checking host runtime…');
    }
    if (!check || check.status === 'skip') {
      return renderAlertBanner({
        variant: 'info',
        message: 'Platform flag enabled — confirm host Apply and Doctor status on /ops before expecting kernel filtering.',
      });
    }
    const variant: 'info' | 'warning' | 'error' = check.status === 'warn' ? 'warning' : 'error';
    if (check.status === 'pass') {
      return el('div', { className: 'settings-edge-xdp__status', 'data-testid': 'edge-xdp-runtime-status' },
        el('div', { className: 'flex items-center gap-2' },
          el('span', { className: 'text-sm font-medium' }, 'Host runtime'),
          renderStatusBadge(check.status ?? '', { kind: 'service', label: displayLabel(check.status ?? '') }),
        ),
        check.message
          ? el('p', { className: 'text-muted text-sm' }, humanizeTechnicalDetail(check.message))
          : null,
      );
    }
    return renderAlertBanner({
      variant,
      message: humanizeTechnicalDetail(check.message ?? 'Edge XDP host runtime not ready') ?? 'Edge XDP host runtime not ready',
    });
  }

  async function handleSave(e: Event) {
    e.preventDefault();
    if (!canWrite) return;
    fieldErrors = {
      tracking_domain: validateTrackingDomain(cfg().tracking_domain ?? ''),
      default_currency: validateCurrency(cfg().default_currency ?? ''),
    };
    if (fieldErrors.tracking_domain || fieldErrors.default_currency) {
      render();
      return;
    }
    saving = true;
    render();
    const stripePatch: PlatformStripeConfig = { enabled: cfg().stripe?.enabled ?? false };
    const secretTrim = stripeSecretInput.trim();
    const webhookTrim = stripeWebhookInput.trim();
    if (secretTrim) stripePatch.secret_key = secretTrim;
    if (webhookTrim) stripePatch.webhook_secret = webhookTrim;

    const patch = {
      tracking_domain: cfg().tracking_domain,
      default_currency: cfg().default_currency,
      timezone: cfg().timezone,
      ingress_schema: cfg().ingress_schema,
      telemetry_enabled: cfg().telemetry_enabled,
      profile: cfg().profile,
      edge_xdp: cfg().edge_xdp,
      network_interface: cfg().network_interface,
      stripe: stripePatch,
    };
    const [savedRes, saveErr] = await to(apiConfirmed('/api/v1/settings/platform', {
      method: 'PATCH',
      body: JSON.stringify(patch),
    }));
    if (saveErr) {
      if (saveErr instanceof ConfirmCancelledError) {
        saving = false;
        render();
        return;
      }
      const v = mapServiceError(saveErr);
      pushToastMessage({ title: v.title, message: v.message, code: v.code });
      saving = false;
      render();
      return;
    }
    const saved = savedRes?.data as PlatformSettingsResponse | undefined;
    if (saved?.restart_required?.length) {
      restartRequired = saved.restart_required;
    }
      pushToastMessage({ title: 'Saved', message: 'Platform settings updated' });
      form = null;
      stripeSecretInput = '';
      stripeWebhookInput = '';
      resource.reload();
      void loadDoctor();
    saving = false;
    render();
  }

  async function handleLicenseApply() {
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
    licenseTokenInput = '';
    const until = (res?.data as { valid_until?: string } | undefined)?.valid_until ?? '';
    pushToastMessage({
      title: 'License updated',
      message: until ? `Valid until ${until}` : 'License applied',
    });
    resource.reload();
  }

  async function handleApply() {
    if (!canWrite) return;
    applying = true;
    render();
    const [, applyErr] = await to(apiConfirmed('/api/v1/settings/platform/apply', { method: 'POST' }));
    if (applyErr) {
      if (applyErr instanceof ConfirmCancelledError) {
        applying = false;
        render();
        return;
      }
      const v = mapServiceError(applyErr);
      pushToastMessage({ title: v.title, message: v.message, code: v.code });
      applying = false;
      render();
      return;
    }
    pushToastMessage({ title: 'Applied', message: 'Configuration written to disk' });
    resource.reload();
    void loadDoctor();
    applying = false;
    render();
  }

  function render() {
    if (destroyed) return;

    if (state.loading && !state.data) {
      replaceChildren(container, el('span', { className: 'text-muted' }, 'Loading…'));
      return;
    }

    if (state.error) {
      replaceChildren(container, renderErrorBlock(state.error));
      return;
    }

    const view = state.data;
    const c = cfg();
    const restartBanner = restartRequired.length > 0 ? restartRequired : (view?.restart_required ?? []);

    replaceChildren(container,
      el('div', { className: 'settings-layout' },
        el('div', { className: 'page-header' },
          el('div', { className: 'page-header__row' },
            el('div', { className: 'flex items-center gap-2' },
              renderIcon('settings', { size: 20, className: 'text-muted' }),
              el('h1', { className: 'page-header__title' }, 'Platform settings'),
            ),
            view?.bootstrap_complete === false
              ? renderStatusBadge('pending', { label: 'Not initialized' })
              : renderStatusBadge('active', { label: 'Initialized', kind: 'service' }),
          ),
          el('p', { className: 'settings-layout__intro' },
            'Configure tracking, deployment profile, edge features, and payment integration. ',
            'Save updates the platform record; Apply writes configuration to disk on the host.',
          ),
        ),
        restartBanner.length > 0
          ? renderAlertBanner({
            variant: 'warning',
            message: `Service restart required: ${restartBanner.map((s: string) => displayLabel(s)).join(', ')}`,
          })
          : null,
        !canWrite
          ? renderAlertBanner({
            variant: 'info',
            message: 'Read-only access — you can view settings but cannot save or apply changes.',
          })
          : null,
        el('div', { className: 'settings-summary' },
          renderSettingsSummaryItem('Deployment profile', displayLabel(c.profile)),
          renderSettingsSummaryItem('Traffic format', displayLabel(c.ingress_schema)),
          renderSettingsSummaryItem('Timezone', c.timezone ?? '—'),
          view?.click_url_template
            ? renderSettingsSummaryItem(
              'Click URL',
              el('code', {
                className: 'settings-summary__code',
                title: view.click_url_template,
              }, view.click_url_template),
            )
            : renderSettingsSummaryItem('Click URL', '—'),
        ),
        el('form', { className: 'settings-form', onSubmit: handleSave },
          el('div', { className: 'settings-grid' },
            renderSectionCard({
              icon: 'globe',
              title: 'General',
              desc: 'Defaults used across campaigns and billing.',
              children: [
                renderFormField({
                  label: 'Tracking domain',
                  error: fieldErrors.tracking_domain,
                  hint: 'Hostname for impression and click tracking',
                  children: el('input', {
                    className: 'form-input',
                    value: c.tracking_domain ?? '',
                    disabled: !canWrite,
                    onInput: (e: Event) => {
                      updateField('tracking_domain', eventTargetValue(e));
                      fieldErrors.tracking_domain = validateTrackingDomain(eventTargetValue(e));
                    },
                  }),
                }),
                renderFormField({
                  label: 'Default currency',
                  error: fieldErrors.default_currency,
                  hint: 'Used for billing and reporting defaults',
                  children: renderSelect({
                    value: (c.default_currency || 'USD').toUpperCase(),
                    options: currencySelectOptions(c.default_currency),
                    disabled: !canWrite,
                    onChange: (v: any) => {
                      updateField('default_currency', v);
                      fieldErrors.default_currency = validateCurrency(v);
                    },
                  }),
                }),
                renderFormField({
                  label: 'Timezone',
                  hint: 'Used for reporting and pacing',
                  children: renderSelect({
                    value: c.timezone || 'UTC',
                    options: timezoneSelectOptions(c.timezone),
                    disabled: !canWrite,
                    onChange: (v: any) => updateField('timezone', v),
                  }),
                }),
              ],
            }),
            renderSectionCard({
              icon: 'server',
              title: 'Deployment',
              desc: 'Runtime profile and network bindings for this installation.',
              children: [
                renderFormField({
                  label: 'Deployment profile',
                  children: renderSelect({
                    value: c.profile ?? 'single_vps',
                    options: PROFILE_SELECT_OPTIONS,
                    disabled: !canWrite,
                    onChange: (v: any) => updateField('profile', v),
                  }),
                }),
                renderFormField({
                  label: 'Traffic format',
                  children: renderSelect({
                    value: c.ingress_schema ?? 'espx_native',
                    options: INGRESS_SELECT_OPTIONS,
                    disabled: !canWrite,
                    onChange: (v: any) => updateField('ingress_schema', v),
                  }),
                }),
                renderFormField({
                  label: 'Network interface',
                  hint: 'Host network interface used by edge acceleration when enabled',
                  children: el('input', {
                    className: 'form-input',
                    value: c.network_interface ?? '',
                    disabled: !canWrite,
                    onInput: (e: Event) => updateField('network_interface', eventTargetValue(e)),
                  }),
                }),
              ],
            }),
            renderSectionCard({
              icon: 'toggle-left',
              title: 'Features',
              desc: 'Optional telemetry and edge acceleration.',
              children: [
                el('div', { className: 'settings-check-group' },
                  renderCheckbox({
                    label: 'Telemetry enabled — export metrics to configured sinks',
                    checked: c.telemetry_enabled ?? false,
                    disabled: !canWrite,
                    onChange: (checked: any) => updateField('telemetry_enabled', checked),
                  }),
                  renderCheckbox({
                    label: 'Edge XDP (Enterprise) — kernel-level IP filtering via installer systemd',
                    checked: c.edge_xdp ?? false,
                    disabled: !canWrite,
                    onChange: (checked: any) => updateField('edge_xdp', checked),
                  }),
                  el('p', { className: 'text-muted text-sm' },
                    'Requires Enterprise license (ebpf_xdp_edge), kernel BTF (6.1+), and installer units ',
                    el('code', null, 'ad-event-processor-edge-xdp'),
                    ' / ',
                    el('code', null, 'ad-event-processor-edge-bpf-sync'),
                    '. Saving updates platform YAML only — use Apply on the host, then verify Doctor on Ops.',
                  ),
                  renderEdgeXdpRuntimeStatus(),
                  renderCheckbox({
                    label: 'Developer mode — show raw sysctl values and API paths in the UI',
                    checked: devModeEnabled(),
                    onChange: (checked: any) => {
                      setDevMode(checked);
                      render();
                    },
                  }),
                ),
              ],
            }),
            renderSectionCard({
              icon: 'credit-card',
              title: 'Stripe',
              desc: 'Payment provider for wallet top-ups. Secrets are stored server-side.',
              children: [
                el('div', { className: 'settings-check-group' },
                  renderCheckbox({
                    label: 'Stripe enabled',
                    checked: c.stripe?.enabled ?? false,
                    disabled: !canWrite,
                    onChange: (checked: any) => updateField('stripe', { ...c.stripe, enabled: checked }),
                  }),
                ),
                renderFormField({
                  label: 'Secret key',
                  htmlFor: 'settings-stripe-secret',
                  hint: 'Leave empty to keep the current key',
                  children: el('input', {
                    id: 'settings-stripe-secret',
                    type: 'password',
                    className: 'form-input',
                    placeholder: 'Enter Stripe secret key',
                    disabled: !canWrite,
                    value: stripeSecretInput,
                    onInput: (e: Event) => { stripeSecretInput = eventTargetValue(e); },
                  }),
                }),
                renderFormField({
                  label: 'Webhook secret',
                  htmlFor: 'settings-stripe-webhook',
                  hint: 'Leave empty to keep the current secret',
                  children: el('input', {
                    id: 'settings-stripe-webhook',
                    type: 'password',
                    className: 'form-input',
                    placeholder: 'Enter webhook signing secret',
                    disabled: !canWrite,
                    value: stripeWebhookInput,
                    onInput: (e: Event) => { stripeWebhookInput = eventTargetValue(e); },
                  }),
                }),
                (view?.secrets?.stripe_secret_key || view?.secrets?.stripe_webhook_secret)
                  ? el('div', { className: 'settings-secrets' },
                    view?.secrets?.stripe_secret_key
                      ? el('div', { className: 'settings-secrets__row' },
                        el('span', null, 'Stored secret key'),
                        el('span', { className: 'font-mono' }, view.secrets.stripe_secret_key),
                      )
                      : null,
                    view?.secrets?.stripe_webhook_secret
                      ? el('div', { className: 'settings-secrets__row' },
                        el('span', null, 'Stored webhook secret'),
                        el('span', { className: 'font-mono' }, view.secrets.stripe_webhook_secret),
                      )
                      : null,
                  )
                  : null,
              ],
            }),
            renderSectionCard({
              icon: 'shield',
              title: 'License renewal',
              desc: 'Offline on-prem: paste the monthly JWT from your vendor. No internet license check.',
              children: [
                renderFormField({
                  label: 'License JWT',
                  hint: 'Paste the full token line from email; applies immediately without restart',
                  children: el('textarea', {
                    className: 'form-input code-block',
                    rows: 4,
                    disabled: !canWrite,
                    value: licenseTokenInput,
                    placeholder: 'eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9...',
                    onInput: (e: Event) => { licenseTokenInput = eventTargetValue(e); },
                  }),
                }),
                canWrite
                  ? renderButton({
                    label: 'Apply license',
                    variant: 'secondary',
                    size: 'sm',
                    icon: 'shield',
                    onClick: handleLicenseApply,
                  })
                  : null,
              ],
            }),
          ),
          canWrite
            ? el('div', { className: 'settings-actions' },
              el('p', { className: 'settings-actions__hint' },
                'Save persists to the database. Apply writes YAML to disk and may require service restart.',
              ),
              el('div', { className: 'settings-actions__buttons' },
                renderButton({
                  label: applying ? 'Applying…' : 'Apply to disk',
                  variant: 'danger',
                  icon: 'shield',
                  loading: applying,
                  disabled: applying,
                  onClick: handleApply,
                }),
                renderButton({
                  label: saving ? 'Saving…' : 'Save',
                  variant: 'primary',
                  icon: 'check',
                  type: 'submit',
                  loading: saving,
                  disabled: saving,
                }),
              ),
            )
            : null,
        ),
        apiKeysSlot,
      ),
    );
    if (!apiKeysPanel && !destroyed) {
      apiKeysPanel = mountCustomerApiKeysPanel(apiKeysSlot, { canCreate: canCreateApiKey });
    }
  }

  const resource = createResource<PlatformSettingsResponse>(
    () => '/api/v1/settings/platform',
    {
      onUpdate: (s) => {
        if (s.error !== lastError) {
          lastError = s.error;
          if (s.error) surfaceServiceErrorToast(s.error);
        }
        Object.assign(state, s);
        if (!doctorSummary && !doctorLoading && !s.loading && !s.error) {
          void loadDoctor();
        }
        render();
      },
    },
  );

  render();

  return {
    destroy() {
      destroyed = true;
      apiKeysPanel?.destroy?.();
      resource.destroy();
    },
  };
}
