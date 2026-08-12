import type { ViewHandle } from '../lib/router_types.js';
import { el, eventTargetValue, eventTargetChecked } from '../lib/dom.js';
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
import { renderButton } from '../ui/button.js';

export type CampaignFiltersPanelOpts = {
  campaignId: string;
  referrerFilter: string;
  canWrite: boolean;
  onSaved?: () => void;
};

/**
 * Mount structured traffic filter editor (no raw Lua).
 *
 * @param {HTMLElement} container
 * @param {CampaignFiltersPanelOpts} opts
 * @returns {{ destroy: () => void }}
 */
export function mountCampaignFiltersPanel(container: HTMLElement, opts: CampaignFiltersPanelOpts): ViewHandle {
  let destroyed = false;
  let rules = parseTrafficFilter(opts.referrerFilter);
  let allowInput = rules.allowReferrers.join(', ');
  let blockInput = rules.blockReferrers.join(', ');
  let blockEmpty = rules.blockEmptyReferrer;
  let saving = false;
  let error: any = null;

  async function save() {
    if (!opts.canWrite) return;
    saving = true;
    error = null;
    render();
    const next = {
      allowReferrers: allowInput.split(',').map((s: any) => s.trim()).filter(Boolean),
      blockReferrers: blockInput.split(',').map((s: any) => s.trim()).filter(Boolean),
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
            onInput: (e: Event) => { allowInput = eventTargetValue(e); },
          }),
        ),
        el('label', { className: 'form-field', htmlFor: 'flt-block' },
          'Block referrers',
          el('input', {
            id: 'flt-block',
            className: 'form-input',
            disabled: !opts.canWrite,
            value: blockInput,
            onInput: (e: Event) => { blockInput = eventTargetValue(e); },
          }),
        ),
        el('label', { className: 'form-check' },
          el('input', {
            type: 'checkbox',
            disabled: !opts.canWrite,
            checked: blockEmpty,
            onChange: (e: Event) => { blockEmpty = eventTargetChecked(e); },
          }),
          ' Block empty referrer',
        ),
        opts.canWrite
          ? el('div', { className: 'cluster--actions' },
            renderButton({
              label: saving ? 'Saving…' : 'Save filters',
              variant: 'primary',
              size: 'sm',
              loading: saving,
              disabled: saving,
              onClick: save,
            }),
            renderButton({
              label: 'Reset',
              variant: 'secondary',
              size: 'sm',
              disabled: saving,
              onClick: reset,
            }),
          )
          : null,
      ),
    );
  }

  render();
  return { destroy() { destroyed = true; } };
}
