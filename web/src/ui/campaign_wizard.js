import { el } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { createCampaign } from '../helpers/campaign_admin_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { isCustomerUuid } from '../helpers/customer_context.js';

/**
 * Open campaign create wizard modal.
 *
 * @param {{
 *   customerId: string,
 *   onCreated: (id: string) => void,
 * }} opts
 * @returns {() => void} destroy
 */
export function openCampaignWizard(opts) {
  const overlay = el('div', {
    className: 'modal-overlay',
    role: 'presentation',
    onClick: (e) => { if (e.target === overlay) close(); },
  });
  const dialog = el('div', {
    className: 'modal',
    role: 'dialog',
    'aria-modal': 'true',
    'aria-labelledby': 'campaign-wizard-title',
  });

  const state = {
    name: '',
    budget: '100.00',
    pacing: 'ASAP',
    timezone: 'UTC',
    countries: 'US',
    busy: false,
    error: null,
  };

  function close() {
    overlay.remove();
    document.removeEventListener('keydown', onKey);
  }

  function onKey(e) {
    if (e.key === 'Escape') close();
  }

  function render() {
    dialog.replaceChildren(
      el('h2', { id: 'campaign-wizard-title', className: 'modal__title' }, 'Create campaign'),
      el('p', { className: 'text-muted text-sm' }, 'Budget is reserved from customer balance on create.'),
      state.error ? el('p', { className: 'text-danger text-sm' }, state.error) : null,
      el('label', { className: 'form-field', htmlFor: 'wiz-name' },
        'Name',
        el('input', {
          id: 'wiz-name',
          className: 'form-input',
          value: state.name,
          onInput: (e) => { state.name = e.target.value; },
        }),
      ),
      el('label', { className: 'form-field', htmlFor: 'wiz-budget' },
        'Budget (USD)',
        el('input', {
          id: 'wiz-budget',
          className: 'form-input font-mono',
          inputMode: 'decimal',
          value: state.budget,
          onInput: (e) => { state.budget = e.target.value; },
        }),
      ),
      el('label', { className: 'form-field', htmlFor: 'wiz-pacing' },
        'Pacing',
        el('select', {
          id: 'wiz-pacing',
          className: 'form-input',
          value: state.pacing,
          onChange: (e) => { state.pacing = e.target.value; },
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
          onInput: (e) => { state.timezone = e.target.value; },
        }),
      ),
      el('label', { className: 'form-field', htmlFor: 'wiz-countries' },
        'Target countries (comma-separated ISO codes)',
        el('input', {
          id: 'wiz-countries',
          className: 'form-input',
          value: state.countries,
          onInput: (e) => { state.countries = e.target.value; },
        }),
      ),
      el('div', { className: 'form-actions' },
        el('button', {
          type: 'button',
          className: 'btn btn--secondary',
          disabled: state.busy,
          onClick: close,
        }, 'Cancel'),
        el('button', {
          type: 'button',
          className: 'btn btn--primary',
          disabled: state.busy,
          onClick: submit,
        }, state.busy ? 'Creating…' : 'Create'),
      ),
    );
  }

  async function submit() {
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
