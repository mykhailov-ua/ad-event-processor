import { el } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { patchCampaign } from '../helpers/campaign_admin_api.js';
import {
  emptyTrafficFilterRules,
  parseTrafficFilter,
  serializeTrafficFilter,
} from '../helpers/traffic_filter.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';

/**
 * Mount structured traffic filter editor (no raw Lua).
 *
 * @param {HTMLElement} container
 * @param {{ campaignId: string, referrerFilter: string, canWrite: boolean, onSaved?: () => void }} opts
 * @returns {{ destroy: () => void }}
 */
export function mountCampaignFiltersPanel(container, opts) {
  let destroyed = false;
  let rules = parseTrafficFilter(opts.referrerFilter);
  let allowInput = rules.allowReferrers.join(', ');
  let blockInput = rules.blockReferrers.join(', ');
  let blockEmpty = rules.blockEmptyReferrer;
  let saving = false;
  let error = null;

  async function save() {
    if (!opts.canWrite) return;
    saving = true;
    error = null;
    render();
    const next = {
      allowReferrers: allowInput.split(',').map((s) => s.trim()).filter(Boolean),
      blockReferrers: blockInput.split(',').map((s) => s.trim()).filter(Boolean),
      blockEmptyReferrer: blockEmpty,
    };
    const [, err] = await to(patchCampaign(opts.campaignId, {
      referrer_filter: serializeTrafficFilter(next),
    }));
    if (destroyed) return;
    saving = false;
    if (err) {
      if (err instanceof ConfirmCancelledError) {
        render();
        return;
      }
      error = mapServiceError(err).message;
      render();
      return;
    }
    rules = next;
    pushToastMessage({ title: 'Filters saved', message: 'Traffic rules updated' });
    opts.onSaved?.();
    render();
  }

  function reset() {
    rules = emptyTrafficFilterRules();
    allowInput = '';
    blockInput = '';
    blockEmpty = false;
    render();
  }

  function render() {
    container.replaceChildren(
      el('div', { className: 'section-card stack' },
        el('h3', { className: 'subsection-title' }, 'Traffic filters'),
        el('p', { className: 'text-muted text-sm' },
          'Structured referrer rules stored as JSON. Hot path reads campaign config after publish.',
        ),
        error ? el('p', { className: 'text-danger text-sm' }, error) : null,
        el('label', { className: 'form-field', htmlFor: 'flt-allow' },
          'Allow referrers (comma-separated hostnames)',
          el('input', {
            id: 'flt-allow',
            className: 'form-input',
            disabled: !opts.canWrite,
            value: allowInput,
            onInput: (e) => { allowInput = e.target.value; },
          }),
        ),
        el('label', { className: 'form-field', htmlFor: 'flt-block' },
          'Block referrers',
          el('input', {
            id: 'flt-block',
            className: 'form-input',
            disabled: !opts.canWrite,
            value: blockInput,
            onInput: (e) => { blockInput = e.target.value; },
          }),
        ),
        el('label', { className: 'form-check' },
          el('input', {
            type: 'checkbox',
            disabled: !opts.canWrite,
            checked: blockEmpty,
            onChange: (e) => { blockEmpty = e.target.checked; },
          }),
          ' Block empty referrer',
        ),
        opts.canWrite
          ? el('div', { className: 'flex gap-2' },
            el('button', {
              type: 'button',
              className: 'btn btn--primary btn--sm',
              disabled: saving,
              onClick: save,
            }, saving ? 'Saving…' : 'Save filters'),
            el('button', {
              type: 'button',
              className: 'btn btn--secondary btn--sm',
              disabled: saving,
              onClick: reset,
            }, 'Reset'),
          )
          : null,
      ),
    );
  }

  render();
  return { destroy() { destroyed = true; } };
}
