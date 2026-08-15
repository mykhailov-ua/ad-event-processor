import type { ViewHandle } from '../lib/router_types.js';
import { el, eventTargetValue, eventTargetChecked } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import { formatMicro } from '../helpers/money.js';
import {
  createMarginGuardPolicy,
  fetchMarginGuardActivity,
  fetchMarginGuardPolicies,
  removeMarginGuardOverride,
} from '../helpers/margin_guard_api.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { tableSkeletonRows, renderEmptyTableCell } from '../ui/data_table.js';
import { renderButton } from '../ui/button.js';

export type CampaignMarginGuardPanelOpts = {
  campaignId: string;
  canWrite: boolean;
};

type MarginGuardPolicy = {
  name?: string;
  min_clicks?: number;
  roi_floor_pct?: number;
  cost_over_revenue_threshold_bps?: number;
  is_active?: boolean;
  [key: string]: unknown;
};

type MarginGuardActivity = {
  created_at?: string;
  placement_id?: string;
  action?: string;
  reason?: string;
  [key: string]: unknown;
};

type MarginSnapshot = {
  advertiser_spend_micro?: number;
  rtb_cost_micro?: number;
  operator_margin_micro?: number;
  publisher_payout_micro?: number;
  threshold_bps?: number;
  margin_breach?: boolean;
};

type MarginGuardForm = {
  name: string;
  min_clicks: string;
  roi_floor_pct: string;
  zero_conv_streak: string;
  cost_over_revenue_threshold_bps: string;
  is_active: boolean;
};

/**
 * Mount margin guard policies, activity, and override controls for a campaign.
 *
 * @param {HTMLElement} container
 * @param {CampaignMarginGuardPanelOpts} opts
 * @returns {{ destroy: () => void, reload: () => void }}
 */
export function mountCampaignMarginGuardPanel(container: HTMLElement, opts: CampaignMarginGuardPanelOpts): ViewHandle & { reload: () => void } {
  let destroyed = false;
  let loading = true;
  let error: any = null;
  let policies: MarginGuardPolicy[] = [];
  let activity: MarginGuardActivity[] = [];
  /** @type {MarginSnapshot|null} */
  let margin: MarginSnapshot | null = null;
  const form: MarginGuardForm = {
    name: 'Default policy',
    min_clicks: '100',
    roi_floor_pct: '0',
    zero_conv_streak: '3',
    cost_over_revenue_threshold_bps: '500',
    is_active: true,
  };
  let saving = false;

  async function load() {
    loading = true;
    error = null;
    render();
    const [polRes, actRes, marginRes] = await Promise.all([
      fetchMarginGuardPolicies(opts.campaignId),
      fetchMarginGuardActivity(opts.campaignId),
      to(api(`/api/v1/campaigns/${opts.campaignId}/margin`)),
    ]);
    if (destroyed) return;
    loading = false;
    policies = (polRes ?? []) as MarginGuardPolicy[];
    activity = (actRes ?? []) as MarginGuardActivity[];
    margin = (marginRes[0]?.data as MarginSnapshot | null | undefined) ?? null;
    render();
  }

  async function savePolicy() {
    if (!opts.canWrite || saving) return;
    saving = true;
    error = null;
    render();
    const body = {
      campaign_id: opts.campaignId,
      name: form.name.trim() || 'Policy',
      min_clicks: Number.parseInt(form.min_clicks, 10) || 0,
      roi_floor_pct: Number.parseFloat(form.roi_floor_pct) || 0,
      zero_conv_streak: Number.parseInt(form.zero_conv_streak, 10) || 0,
      cost_over_revenue_threshold_bps: Number.parseInt(form.cost_over_revenue_threshold_bps, 10) || 500,
      is_active: form.is_active,
    };
    const [, err] = await to(createMarginGuardPolicy(body));
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
    pushToastMessage({ title: 'Policy saved', message: body.name });
    load();
  }

  async function clearOverride(placementId: any) {
    const [, err] = await to(removeMarginGuardOverride(opts.campaignId, placementId));
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Override failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({ title: 'Override removed', message: placementId });
    load();
  }

  function render() {
    container.replaceChildren(
      el('div', { className: 'stack' },
        loading ? el('p', { className: 'text-muted' }, 'Loading margin guard…') : null,
        error ? el('p', { className: 'text-danger text-sm' }, error) : null,
        margin
          ? el('div', { className: 'metric-row section-block' },
            el('div', { className: 'metric-card' },
              el('div', { className: 'metric-card__label' }, '1h spend'),
              el('div', { className: 'metric-card__value font-mono' },
                '$' + formatMicro(margin.advertiser_spend_micro ?? 0),
              ),
            ),
            el('div', { className: 'metric-card' },
              el('div', { className: 'metric-card__label' }, 'RTB cost'),
              el('div', { className: 'metric-card__value font-mono' },
                '$' + formatMicro(margin.rtb_cost_micro ?? 0),
              ),
            ),
            el('div', { className: 'metric-card' },
              el('div', { className: 'metric-card__label' }, 'Operator margin'),
              el('div', { className: 'metric-card__value font-mono' },
                '$' + formatMicro(margin.operator_margin_micro ?? 0),
              ),
            ),
            el('div', { className: 'metric-card' },
              el('div', { className: 'metric-card__label' }, 'Publisher payout'),
              el('div', { className: 'metric-card__value font-mono' },
                '$' + formatMicro(margin.publisher_payout_micro ?? 0),
              ),
            ),
            el('div', { className: 'metric-card' },
              el('div', { className: 'metric-card__label' }, 'Threshold (bps)'),
              el('div', { className: 'metric-card__value' }, String(margin.threshold_bps ?? '—')),
            ),
            el('div', { className: 'metric-card' },
              el('div', { className: 'metric-card__label' }, 'Breach'),
              el('div', {
                className: 'metric-card__value' + (margin.margin_breach ? ' text-danger' : ''),
              }, margin.margin_breach ? 'Yes' : 'No'),
            ),
          )
          : null,
        el('div', { className: 'section-card stack' },
          el('h3', { className: 'subsection-title' }, 'Policies'),
          el('div', { className: 'table-wrapper' },
            el('table', { className: 'data-table', 'aria-label': 'Margin guard policies' },
              el('thead', null,
                el('tr', null,
                  el('th', { scope: 'col' }, 'Name'),
                  el('th', { scope: 'col' }, 'Min clicks'),
                  el('th', { scope: 'col' }, 'ROI floor %'),
                  el('th', { scope: 'col' }, 'Cost/rev bps'),
                  el('th', { scope: 'col' }, 'Active'),
                ),
              ),
              el('tbody', null,
                loading ? tableSkeletonRows(5) : null,
                !loading && policies.length === 0
                  ? el('tr', null, el('td', { colSpan: 5 }, 'No policies yet.'))
                  : null,
                policies.map((p: any) => el('tr', null,
                  el('td', null, p.name ?? '—'),
                  el('td', null, String(p.min_clicks ?? 0)),
                  el('td', null, String(p.roi_floor_pct ?? 0)),
                  el('td', null, String(p.cost_over_revenue_threshold_bps ?? 0)),
                  el('td', null, p.is_active ? 'Yes' : 'No'),
                )),
              ),
            ),
          ),
          opts.canWrite
            ? el('div', { className: 'stack mt-4' },
              el('h4', { className: 'subsection-title' }, 'Add policy'),
              el('label', { className: 'form-field', htmlFor: 'mg-name' },
                'Name',
                el('input', {
                  id: 'mg-name',
                  className: 'form-input form-input--sm',
                  value: form.name,
                  onInput: (e: Event) => { form.name = eventTargetValue(e); },
                }),
              ),
              el('div', { className: 'form-row' },
                el('label', { className: 'form-field', htmlFor: 'mg-min-clicks' },
                  'Min clicks',
                  el('input', {
                    id: 'mg-min-clicks',
                    className: 'form-input form-input--sm',
                    inputMode: 'numeric',
                    value: form.min_clicks,
                    onInput: (e: Event) => { form.min_clicks = eventTargetValue(e); },
                  }),
                ),
                el('label', { className: 'form-field', htmlFor: 'mg-roi' },
                  'ROI floor %',
                  el('input', {
                    id: 'mg-roi',
                    className: 'form-input form-input--sm',
                    inputMode: 'decimal',
                    value: form.roi_floor_pct,
                    onInput: (e: Event) => { form.roi_floor_pct = eventTargetValue(e); },
                  }),
                ),
              ),
              el('div', { className: 'form-row' },
                el('label', { className: 'form-field', htmlFor: 'mg-streak' },
                  'Zero-conv streak',
                  el('input', {
                    id: 'mg-streak',
                    className: 'form-input form-input--sm',
                    inputMode: 'numeric',
                    value: form.zero_conv_streak,
                    onInput: (e: Event) => { form.zero_conv_streak = eventTargetValue(e); },
                  }),
                ),
                el('label', { className: 'form-field', htmlFor: 'mg-bps' },
                  'Cost over revenue (bps)',
                  el('input', {
                    id: 'mg-bps',
                    className: 'form-input form-input--sm',
                    inputMode: 'numeric',
                    value: form.cost_over_revenue_threshold_bps,
                    onInput: (e: Event) => { form.cost_over_revenue_threshold_bps = eventTargetValue(e); },
                  }),
                ),
              ),
              el('label', { className: 'form-check' },
                el('input', {
                  type: 'checkbox',
                  checked: form.is_active,
                  onChange: (e: Event) => { form.is_active = eventTargetChecked(e); },
                }),
                ' Active',
              ),
              renderButton({
                label: saving ? 'Saving…' : 'Create policy',
                variant: 'primary',
                size: 'sm',
                loading: saving,
                disabled: saving,
                onClick: savePolicy,
              }),
            )
            : null,
        ),
        el('div', { className: 'section-card stack' },
          el('h3', { className: 'subsection-title' }, 'Activity'),
          el('div', { className: 'table-wrapper' },
            el('table', { className: 'data-table', 'aria-label': 'Margin guard activity' },
              el('thead', null,
                el('tr', null,
                  el('th', { scope: 'col' }, 'Time'),
                  el('th', { scope: 'col' }, 'Placement'),
                  el('th', { scope: 'col' }, 'Action'),
                  el('th', { scope: 'col' }, 'Reason'),
                  el('th', { scope: 'col' }, ''),
                ),
              ),
              el('tbody', null,
                loading ? tableSkeletonRows(5) : null,
                !loading && activity.length === 0
                  ? el('tr', null,
                    renderEmptyTableCell(5, {
                      title: 'No guard actions yet',
                      description: 'Margin guard activity appears after policies trigger.',
                      icon: 'shield',
                    }),
                  )
                  : null,
                activity.map((row: any) => el('tr', null,
                  el('td', null,
                    row.created_at ? new Date(row.created_at).toLocaleString() : '—',
                  ),
                  el('td', { className: 'font-mono text-hint' }, row.placement_id ?? '—'),
                  el('td', null, row.action ?? '—'),
                  el('td', null, row.reason ?? '—'),
                  el('td', null,
                    opts.canWrite && row.action === 'pause' && row.placement_id
                      ? renderButton({
                        label: 'Remove override',
                        variant: 'secondary',
                        size: 'sm',
                        onClick: () => clearOverride(row.placement_id),
                      })
                      : null,
                  ),
                )),
              ),
            ),
          ),
        ),
      ),
    );
  }

  load();
  return {
    destroy() { destroyed = true; },
    reload: load,
  };
}
