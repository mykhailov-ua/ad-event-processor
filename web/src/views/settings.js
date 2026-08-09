import { el, replaceChildren } from '../lib/dom.js';
import { createResource } from '../lib/fetch_resource.js';
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
import { renderIcon } from '../ui/icon.js';
import {
  currencySelectOptions,
  displayLabel,
  INGRESS_SELECT_OPTIONS,
  PROFILE_SELECT_OPTIONS,
  timezoneSelectOptions,
} from '../helpers/display_labels.js';
import { devModeEnabled, setDevMode } from '../helpers/dev_mode.js';

/**
 * Mount the platform settings view with save and apply actions.
 *
 * @param {HTMLElement} container
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container) {
  let destroyed = false;
  let form = null;
  let saving = false;
  let applying = false;
  let restartRequired = [];
  let lastError = null;
  let stripeSecretInput = '';
  let stripeWebhookInput = '';
  let licenseTokenInput = '';

  const user = auth.getUser();
  const canWrite = can(user?.permissions ?? [], 'settings:write');
  const canCreateApiKey = can(user?.permissions ?? [], 'campaigns:write');
  const apiKeysSlot = el('div');
  /** @type {{ destroy: () => void }|null} */
  let apiKeysPanel = null;

  const state = { data: null, loading: true, error: null };
  let fieldErrors = {};

  function cfg() {
    return form ?? state.data?.config ?? {};
  }

  function updateField(key, value) {
    form = { ...cfg(), [key]: value };
    render();
  }

  async function handleSave(e) {
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
    const stripePatch = { enabled: cfg().stripe?.enabled ?? false };
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
    const saved = savedRes?.data;
    if (saved?.restart_required?.length) {
      restartRequired = saved.restart_required;
    }
      pushToastMessage({ title: 'Saved', message: 'Platform settings updated' });
      form = null;
      stripeSecretInput = '';
      stripeWebhookInput = '';
      resource.reload();
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
    const until = res?.data?.valid_until ?? '';
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
            message: `Service restart required: ${restartBanner.map((s) => displayLabel(s)).join(', ')}`,
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
                    onInput: (e) => {
                      updateField('tracking_domain', e.target.value);
                      fieldErrors.tracking_domain = validateTrackingDomain(e.target.value);
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
                    onChange: (v) => {
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
                    onChange: (v) => updateField('timezone', v),
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
                    onChange: (v) => updateField('profile', v),
                  }),
                }),
                renderFormField({
                  label: 'Traffic format',
                  children: renderSelect({
                    value: c.ingress_schema ?? 'espx_native',
                    options: INGRESS_SELECT_OPTIONS,
                    disabled: !canWrite,
                    onChange: (v) => updateField('ingress_schema', v),
                  }),
                }),
                renderFormField({
                  label: 'Network interface',
                  hint: 'Host network interface used by edge acceleration when enabled',
                  children: el('input', {
                    className: 'form-input',
                    value: c.network_interface ?? '',
                    disabled: !canWrite,
                    onInput: (e) => updateField('network_interface', e.target.value),
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
                    onChange: (checked) => updateField('telemetry_enabled', checked),
                  }),
                  renderCheckbox({
                    label: 'Edge acceleration (XDP) — fast network filtering at the kernel',
                    checked: c.edge_xdp ?? false,
                    disabled: !canWrite,
                    onChange: (checked) => updateField('edge_xdp', checked),
                  }),
                  renderCheckbox({
                    label: 'Developer mode — show raw sysctl values and API paths in the UI',
                    checked: devModeEnabled(),
                    onChange: (checked) => {
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
                    onChange: (checked) => updateField('stripe', { ...c.stripe, enabled: checked }),
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
                    onInput: (e) => { stripeSecretInput = e.target.value; },
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
                    onInput: (e) => { stripeWebhookInput = e.target.value; },
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
                    className: 'form-input',
                    rows: 4,
                    disabled: !canWrite,
                    value: licenseTokenInput,
                    placeholder: 'eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9...',
                    onInput: (e) => { licenseTokenInput = e.target.value; },
                  }),
                }),
                canWrite
                  ? el('button', {
                    type: 'button',
                    className: 'btn btn--secondary',
                    onClick: handleLicenseApply,
                  },
                    renderIcon('shield', { size: 14 }),
                    'Apply license',
                  )
                  : null,
              ],
            }),
          ),
          canWrite
            ? el('div', { className: 'settings-actions' },
              el('p', { className: 'settings-actions__hint' },
                'Save persists to the database. Apply writes YAML to disk and may require service restart.',
              ),
              el('button', {
                type: 'button',
                className: 'btn btn--danger',
                disabled: applying,
                onClick: handleApply,
              },
                renderIcon('shield', { size: 14 }),
                applying ? 'Applying…' : 'Apply to disk',
              ),
              el('button', {
                type: 'submit',
                className: 'btn btn--primary',
                disabled: saving,
              },
                renderIcon('check', { size: 14 }),
                saving ? 'Saving…' : 'Save',
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

  const resource = createResource(
    () => '/api/v1/settings/platform',
    {
      onUpdate: (s) => {
        if (s.error !== lastError) {
          lastError = s.error;
          if (s.error) surfaceServiceErrorToast(s.error);
        }
        Object.assign(state, s);
        render();
      },
    },
  );

  render();

  return {
    destroy() {
      destroyed = true;
      apiKeysPanel?.destroy();
      resource.destroy();
    },
  };
}
