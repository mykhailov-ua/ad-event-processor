import { el } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { createCampaign, patchCampaign } from '../helpers/campaign_admin_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { isCustomerUuid } from '../helpers/customer_context.js';
import { renderButton } from './button.js';

export type CampaignWizardOpts = {
  customerId: string;
  onCreated: (id: string) => void;
};

type WizardState = {
  name: string;
  budget: string;
  pacing: string;
  timezone: string;
  countries: string;
  targetUrl: string;
  busy: boolean;
  error: string | null;
};

/**
 * Open campaign create wizard modal.
 */
export function openCampaignWizard(opts: CampaignWizardOpts): () => void {
  const overlay = el('div', {
    className: 'modal-overlay',
    role: 'presentation',
    onClick: (e: Event) => { if (e.target === overlay) close(); },
  });
  const dialog = el('div', {
    className: 'modal',
    role: 'dialog',
    'aria-modal': 'true',
    'aria-labelledby': 'campaign-wizard-title',
  });

  const state: WizardState = {
    name: '',
    budget: '100.00',
    pacing: 'ASAP',
    timezone: 'UTC',
    countries: 'US',
    targetUrl: '',
    busy: false,
    error: null,
  };

  function close(): void {
    overlay.remove();
    document.removeEventListener('keydown', onKey);
  }

  function onKey(e: KeyboardEvent): void {
    if (e.key === 'Escape') close();
  }

  function render(): void {
    const nodes: Array<Node | string> = [
      el('h2', { id: 'campaign-wizard-title', className: 'modal__title' }, 'Create campaign'),
      el('p', { className: 'text-muted text-sm' }, 'Budget is reserved from customer balance on create.'),
    ];
    if (state.error) nodes.push(el('p', { className: 'text-danger text-sm' }, state.error));
    nodes.push(
      el('label', { className: 'form-field', htmlFor: 'wiz-name' },
        'Name',
        el('input', {
          id: 'wiz-name',
          className: 'form-input',
          value: state.name,
          onInput: (e: Event) => { state.name = (e.target as HTMLInputElement).value; },
        }),
      ),
      el('label', { className: 'form-field', htmlFor: 'wiz-budget' },
        'Budget (USD)',
        el('input', {
          id: 'wiz-budget',
          className: 'form-input font-mono',
          inputMode: 'decimal',
          value: state.budget,
          onInput: (e: Event) => { state.budget = (e.target as HTMLInputElement).value; },
        }),
      ),
      el('label', { className: 'form-field', htmlFor: 'wiz-pacing' },
        'Pacing',
        el('select', {
          id: 'wiz-pacing',
          className: 'form-input',
          value: state.pacing,
          onChange: (e: Event) => { state.pacing = (e.target as HTMLSelectElement).value; },
        },
          el('option', { value: 'ASAP' }, 'ASAP'),
          el('option', { value: 'EVEN' }, 'Even'),
        ),
      ),
      el('label', { className: 'form-field', htmlFor: 'wiz-tz' },
        'Timezone',
        el('input', {
          id: 'wiz-tz',
          className: 'form-input',
          value: state.timezone,
          onInput: (e: Event) => { state.timezone = (e.target as HTMLInputElement).value; },
        }),
      ),
      el('label', { className: 'form-field', htmlFor: 'wiz-countries' },
        'Target countries (comma-separated ISO codes)',
        el('input', {
          id: 'wiz-countries',
          className: 'form-input',
          value: state.countries,
          onInput: (e: Event) => { state.countries = (e.target as HTMLInputElement).value; },
        }),
      ),
      el('label', { className: 'form-field', htmlFor: 'wiz-target-url' },
        'Landing URL (optional)',
        el('input', {
          id: 'wiz-target-url',
          className: 'form-input',
          type: 'url',
          placeholder: 'https://example.com/offer',
          value: state.targetUrl,
          onInput: (e: Event) => { state.targetUrl = (e.target as HTMLInputElement).value; },
        }),
      ),
      el('div', { className: 'form-actions cluster--actions' },
        renderButton({
          label: 'Cancel',
          variant: 'secondary',
          disabled: state.busy,
          onClick: close,
        }),
        renderButton({
          label: state.busy ? 'Creating…' : 'Create',
          variant: 'primary',
          loading: state.busy,
          disabled: state.busy,
          onClick: () => { void submit(); },
        }),
      ),
    );
    dialog.replaceChildren(...nodes);
  }

  async function submit(): Promise<void> {
    if (!isCustomerUuid(opts.customerId)) {
      state.error = 'Valid customer UUID required';
      render();
      return;
    }
    const budget = Number.parseFloat(state.budget);
    if (!state.name.trim() || !Number.isFinite(budget) || budget <= 0) {
      state.error = 'Name and positive budget are required';
      render();
      return;
    }
    state.busy = true;
    state.error = null;
    render();
    const countries = state.countries.split(',').map((c) => c.trim().toUpperCase()).filter(Boolean);
    const [data, err] = await to(createCampaign(opts.customerId, {
      name: state.name.trim(),
      budget_limit: budget,
      pacing_mode: state.pacing,
      timezone: state.timezone.trim() || 'UTC',
      target_countries: countries,
    }));
    if (err) {
      if (err instanceof ConfirmCancelledError) {
        state.busy = false;
        render();
        return;
      }
      const view = mapServiceError(err);
      state.error = view.message;
      state.busy = false;
      render();
      return;
    }
    pushToastMessage({ title: 'Campaign created', message: data?.id ?? 'OK' });
    const landing = state.targetUrl.trim();
    if (data?.id && landing) {
      const [, patchErr] = await to(patchCampaign(data.id, { target_url: landing }));
      if (patchErr) {
        pushToastMessage({
          title: 'Landing URL',
          message: patchErr.message ?? 'Campaign created; landing URL not saved',
        });
      }
    }
    close();
    if (data?.id) opts.onCreated(data.id);
  }

  render();
  overlay.appendChild(dialog);
  document.body.appendChild(overlay);
  document.addEventListener('keydown', onKey);
  const nameInput = dialog.querySelector('#wiz-name');
  if (nameInput instanceof HTMLElement) nameInput.focus();

  return close;
}
