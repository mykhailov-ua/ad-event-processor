import type { ViewHandle } from '../lib/router_types.js';
import { el, replaceChildren, eventTargetValue } from '../lib/dom.js';
import { to } from '../lib/to.js';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderSectionCard } from '../ui/section_card.js';
import { renderStatusBadge } from '../ui/status_badge.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { openRTBRoutingHint } from '../helpers/openrtb_endpoint.js';
import {
  fetchRtbIntegrationProfile,
  fetchRtbShadowDiff,
  validateBidRequest,
  applyRtbFloors,
  fetchRtbReconcileExport,
} from '../helpers/rtb_api.js';
import type { RtbFloorsApplyResult, RtbFloorSuggestionDTO, RtbReconcileExportDTO } from '../types/api/rtb.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import { formatAmountMicro } from '../helpers/money.js';
import { VALIDATE_BID_FIXTURE } from '../helpers/openrtb_endpoint.js';
import { renderButton } from '../ui/button.js';

const SMOKE_FIXTURE = VALIDATE_BID_FIXTURE;

type RtbRuntimeHints = {
  rtb_mode?: string;
  rtb_enabled?: boolean;
  rtb_exchange_no_bid_mode?: string;
  [key: string]: unknown;
};

type RtbEndpoints = {
  openrtb_bid_url?: string;
  edge_expose_openrtb?: boolean;
  edge_port_hint?: string;
  tracker_port_hint?: string;
  [key: string]: unknown;
};

type RtbIntegrationProfile = {
  openrtb_version?: string;
  required?: string[];
  supported?: string[];
  not_supported?: string[];
  runtime?: RtbRuntimeHints;
  endpoints?: RtbEndpoints;
  [key: string]: unknown;
};

type RtbShadowDiff = {
  source?: string;
  window?: string;
  shadow_evals?: number;
  parity_rate?: number;
  mismatch_rate?: number;
  shadow_no_bid?: number;
  [key: string]: unknown;
};

type ValidateBidResult = {
  valid?: boolean;
  errors?: string[];
  [key: string]: unknown;
};

/**
 * Copy text helper.
 *
 * @param {string} label
 * @param {string} text
 */
function copyText(label: string, text: string) {
  navigator.clipboard?.writeText(text).then(() => {
    pushToastMessage({ title: 'Copied', message: `${label} copied` });
  }).catch(() => {
    pushToastMessage({ title: 'Copy failed', message: text });
  });
}

/**
 * @param {string} label
 * @param {string} value
 * @returns {HTMLElement}
 */
function copyRow(label: string, value: string) {
  return el('div', { className: 'integration-copy-row' },
    el('div', { className: 'integration-copy-row__head' },
      el('span', { className: 'form-label' }, label),
      renderButton({
        label: 'Copy',
        variant: 'secondary',
        size: 'sm',
        onClick: () => copyText(label, value),
      }),
    ),
    el('code', { className: 'code-block' }, value),
  );
}

/**
 * @param {string[]} items
 * @returns {HTMLElement}
 */
function renderBulletList(items: string[]) {
  const ul = el('ul', { className: 'plain-list' });
  for (let i = 0; i < items.length; i++) {
    ul.appendChild(el('li', { className: 'plain-list__item font-mono text-sm' }, items[i]));
  }
  return ul;
}

/**
 * Mount RTB exchange integration onboarding view.
 *
 * @param {HTMLElement} container
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container: HTMLElement): ViewHandle {
  let destroyed = false;
  const user = auth.getUser();
  const canWrite = can(user?.permissions ?? [], 'rtb:write');
  const canApplyFloors = can(user?.permissions ?? [], 'settings:write');

  let loading = true;
  /** @type {Error|null} */
  let error: Error | string | null = null;
  /** @type {object|null} */
  let profile: RtbIntegrationProfile | null = null;
  /** @type {object|null} */
  let shadow: RtbShadowDiff | null = null;
  let validateBusy = false;
  /** @type {object|null} */
  let validateResult: ValidateBidResult | null = null;
  /** @type {Error|null} */
  let validateError: Error | string | null = null;
  let fixtureText = JSON.stringify(SMOKE_FIXTURE, null, 2);
  let floorsBusy = false;
  let floorsResult: RtbFloorsApplyResult | null = null;
  let floorsError: string | null = null;
  let reconcileBusy = false;
  let reconcileData: RtbReconcileExportDTO | null = null;
  let reconcileError: string | null = null;

  async function runFloors(dryRun: boolean): Promise<void> {
    if (!canApplyFloors || floorsBusy) return;
    floorsBusy = true;
    floorsError = null;
    if (dryRun) floorsResult = null;
    render();
    const [res, err] = await to(applyRtbFloors(dryRun));
    floorsBusy = false;
    if (destroyed) return;
    if (err) {
      if (err instanceof ConfirmCancelledError) {
        render();
        return;
      }
      const view = mapServiceError(err);
      floorsError = view.message;
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      render();
      return;
    }
    floorsResult = res as RtbFloorsApplyResult;
    if (!dryRun) {
      pushToastMessage({
        title: 'Floors applied',
        message: `${floorsResult.applied} deal(s), ${floorsResult.outbox_rows} outbox row(s)`,
      });
    }
    render();
  }

  async function loadReconcileExport(): Promise<void> {
    if (reconcileBusy) return;
    reconcileBusy = true;
    reconcileError = null;
    render();
    const [res, err] = await to(fetchRtbReconcileExport('24h'));
    reconcileBusy = false;
    if (destroyed) return;
    if (err) {
      const view = mapServiceError(err);
      reconcileError = view.message;
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      render();
      return;
    }
    reconcileData = res as RtbReconcileExportDTO;
    const json = JSON.stringify(reconcileData, null, 2);
    const blob = new Blob([json], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = 'rtb-reconcile-export.json';
    anchor.click();
    URL.revokeObjectURL(url);
    pushToastMessage({
      title: 'Reconcile export',
      message: `${reconcileData.bids} bids, ${reconcileData.wins} wins`,
    });
    render();
  }

  function renderReconcilePanel(): HTMLElement {
    return renderSectionCard({
      title: 'Reconcile export',
      icon: 'download',
      desc: 'Download ClickHouse bid/win snapshot and live-gate readiness for the last 24h.',
      children: [
        renderButton({
          label: reconcileBusy ? 'Exporting…' : 'Download reconcile JSON',
          variant: 'secondary',
          size: 'sm',
          loading: reconcileBusy,
          disabled: reconcileBusy,
          testId: 'rtb-reconcile-export',
          onClick: loadReconcileExport,
        }),
        reconcileError ? el('p', { className: 'text-danger text-sm' }, reconcileError) : null,
        reconcileData
          ? el('dl', { className: 'definition-list mt-2', 'data-testid': 'rtb-reconcile-summary' },
            el('dt', null, 'Bids'),
            el('dd', null, String(reconcileData.bids)),
            el('dt', null, 'Wins'),
            el('dd', null, String(reconcileData.wins)),
            el('dt', null, 'Spend (micro)'),
            el('dd', { className: 'font-mono' }, String(reconcileData.spend_micro)),
            el('dt', null, 'Live gate'),
            el('dd', null, reconcileData.live_gate_ready ? 'Ready' : 'Not ready'),
          )
          : null,
      ],
    });
  }

  function renderFloorsPanel(): HTMLElement {
    const suggestions = floorsResult?.suggestions ?? [];
    return renderSectionCard({
      title: 'Bid floor optimizer',
      icon: 'trending-up',
      desc: 'Preview ClickHouse-backed floor suggestions, then apply to deals (outbox propagation).',
      children: [
        canApplyFloors
          ? el('div', { className: 'cluster--actions mb-3' },
            renderButton({
              label: floorsBusy ? 'Working…' : 'Preview',
              variant: 'secondary',
              size: 'sm',
              loading: floorsBusy,
              disabled: floorsBusy,
              testId: 'rtb-floors-preview',
              onClick: () => runFloors(true),
            }),
            renderButton({
              label: 'Apply floors',
              variant: 'danger',
              size: 'sm',
              loading: floorsBusy,
              disabled: floorsBusy,
              testId: 'rtb-floors-apply',
              onClick: () => runFloors(false),
            }),
          )
          : el('p', { className: 'text-muted text-sm' }, 'settings:write required to preview or apply floors.'),
        floorsError ? el('p', { className: 'text-danger text-sm' }, floorsError) : null,
        suggestions.length > 0
          ? el('div', { className: 'table-wrapper' },
            el('table', { className: 'data-table' },
              el('thead', null,
                el('tr', null,
                  el('th', { scope: 'col' }, 'Placement'),
                  el('th', { scope: 'col' }, 'Deal'),
                  el('th', { scope: 'col' }, 'Current'),
                  el('th', { scope: 'col' }, 'Suggested'),
                  el('th', { scope: 'col' }, 'Win rate'),
                ),
              ),
              el('tbody', null,
                suggestions.map((row: RtbFloorSuggestionDTO) => el('tr', null,
                  el('td', { className: 'font-mono text-xs' }, row.placement_id),
                  el('td', { className: 'font-mono text-xs' }, row.deal_id),
                  el('td', { className: 'font-mono' }, formatAmountMicro(row.current_floor_micro)),
                  el('td', { className: 'font-mono' }, formatAmountMicro(row.suggested_floor_micro)),
                  el('td', null, `${(row.win_rate * 100).toFixed(1)}%`),
                )),
              ),
            ),
          )
          : floorsResult?.dry_run
            ? el('p', { className: 'text-muted text-sm' }, 'Preview returned no suggestions.')
            : null,
      ],
    });
  }

  async function load() {
    loading = true;
    error = null;
    render();
    const [profRes, shadowRes] = await Promise.all([
      to(fetchRtbIntegrationProfile()),
      to(fetchRtbShadowDiff('1h')),
    ]);
    if (destroyed) return;
    loading = false;
    if (profRes[1]) {
      error = profRes[1];
      render();
      return;
    }
    profile = (profRes[0] ?? null) as RtbIntegrationProfile | null;
    shadow = shadowRes[1] ? null : (shadowRes[0] as RtbShadowDiff | null);
    render();
  }

  async function runValidate() {
    validateBusy = true;
    validateError = null;
    validateResult = null;
    render();
    let body;
    try {
      body = JSON.parse(fixtureText);
    } catch (err) {
      validateBusy = false;
      const msg = err instanceof Error ? err.message : String(err);
      validateError = new Error(`Invalid JSON: ${msg}`);
      render();
      return;
    }
    const [res, err] = await to(validateBidRequest(body));
    if (destroyed) return;
    validateBusy = false;
    if (err) {
      validateError = err;
    } else {
      validateResult = res as ValidateBidResult | null;
    }
    render();
  }

  function renderRuntimeHints(runtime: RtbRuntimeHints | null | undefined) {
    const mode = (runtime?.rtb_mode || 'off').toLowerCase();
    const enabled = runtime?.rtb_enabled === true;
    return el('dl', { className: 'definition-list' },
      el('dt', null, 'RTB_MODE'),
      el('dd', { className: 'font-mono' }, enabled ? mode : 'off (RTB disabled)'),
      el('dt', null, 'RTB_EXCHANGE_NO_BID_MODE'),
      el('dd', { className: 'font-mono' }, runtime?.rtb_exchange_no_bid_mode || '—'),
      el('dt', null, 'Note'),
      el('dd', { className: 'text-muted text-sm' },
        'Env-driven on tracker/control. RTB_MODE=shadow|live affects in-auction path on /track; exchange partners use POST /openrtb/bid below.',
      ),
    );
  }

  function renderShadowSummary(data: RtbShadowDiff | null) {
    if (!data || data.source === 'unavailable') {
      return el('p', { className: 'text-muted text-sm' },
        'Shadow-diff metrics are recorded on tracker nodes. Control plane shows unavailable when not wired to hot-path counters.',
      );
    }
    const parity = data.parity_rate != null ? `${(data.parity_rate * 100).toFixed(1)}%` : '—';
    const mismatch = data.mismatch_rate != null ? `${(data.mismatch_rate * 100).toFixed(1)}%` : '—';
    return el('dl', { className: 'definition-list', 'data-testid': 'rtb-shadow-diff' },
      el('dt', null, 'Window'),
      el('dd', { className: 'font-mono' }, data.window ?? '1h'),
      el('dt', null, 'Shadow evals'),
      el('dd', null, String(data.shadow_evals ?? 0)),
      el('dt', null, 'Parity rate'),
      el('dd', null, parity),
      el('dt', null, 'Mismatch rate'),
      el('dd', null, mismatch),
      el('dt', null, 'Shadow no-bid'),
      el('dd', null, String(data.shadow_no_bid ?? 0)),
    );
  }

  function render() {
    if (destroyed) return;
    if (error) {
      replaceChildren(container, renderErrorBlock(error));
      return;
    }

    const endpoints = profile?.endpoints ?? {};
    const bidURL = endpoints.openrtb_bid_url ?? 'https://track.example/openrtb/bid';
    const routing = openRTBRoutingHint({
      edgeExposeOpenRTB: endpoints.edge_expose_openrtb,
      edgePortHint: endpoints.edge_port_hint,
      trackerPortHint: endpoints.tracker_port_hint,
    });

    const children = [
      el('div', { className: 'page-header' },
        el('h1', { className: 'page-header__title' }, 'RTB integration'),
        el('p', { className: 'page-header__desc' },
          'OpenRTB 2.6 exchange onboarding — profile, env hints, validate-bid smoke, shadow parity.',
        ),
        el('p', { className: 'text-muted text-sm' },
          el('a', { href: '/rtb/deals' }, '← PMP deals'),
        ),
      ),
    ];

    if (loading) {
      children.push(el('p', { className: 'loading-hint' }, 'Loading integration profile…'));
      replaceChildren(container, el('section', { 'data-testid': 'rtb-integration-view' }, ...children));
      return;
    }

    const supported = (profile?.supported ?? []) as string[];
    children.push(
      renderSectionCard({
        title: 'SSP endpoint',
        icon: 'globe',
        desc: 'Point exchange partners at this URL (chunked POST body allowed).',
        children: [
          copyRow('POST /openrtb/bid', bidURL),
          el('p', { className: 'text-muted text-sm' }, `Routing: ${routing}`),
          endpoints.edge_expose_openrtb
            ? renderStatusBadge('ok', { label: 'Edge OpenRTB enabled' })
            : renderStatusBadge('neutral', { label: 'Tracker ports only — enable edge in Platform settings' }),
        ],
      }),
      renderSectionCard({
        title: 'Runtime exchange settings',
        icon: 'settings',
        desc: 'Read-only env hints from this deployment (not editable in UI).',
        children: [renderRuntimeHints(profile?.runtime)],
      }),
      renderSectionCard({
        title: 'Integration profile',
        icon: 'clipboard-text',
        desc: `OpenRTB ${profile?.openrtb_version ?? '2.6'} — required fields and capability matrix.`,
        children: [
          el('h3', { className: 'subsection-title' }, 'Required'),
          renderBulletList((profile?.required ?? []) as string[]),
          el('h3', { className: 'subsection-title' }, 'Supported'),
          renderBulletList(supported.slice(0, 24)),
          supported.length > 24
            ? el('p', { className: 'text-muted text-xs' }, `+ ${supported.length - 24} more fields`)
            : null,
          el('h3', { className: 'subsection-title' }, 'Not supported'),
          renderBulletList((profile?.not_supported ?? []) as string[]),
        ],
      }),
      renderSectionCard({
        title: 'Validate bid request',
        icon: 'check',
        desc: 'POST smoke test against control-plane validator (same rules as tracker exchange).',
        children: [
          el('textarea', {
            className: 'form-input code-block',
            rows: 14,
            value: fixtureText,
            onInput: (e: Event) => { fixtureText = eventTargetValue(e); },
          }),
          el('div', { className: 'cluster--actions mt-2' },
            renderButton({
              label: validateBusy ? 'Validating…' : 'Run validate-bid',
              variant: 'primary',
              loading: validateBusy,
              disabled: validateBusy,
              onClick: runValidate,
            }),
            renderButton({
              label: 'Reset fixture',
              variant: 'secondary',
              disabled: validateBusy,
              onClick: () => {
                fixtureText = JSON.stringify(SMOKE_FIXTURE, null, 2);
                render();
              },
            }),
          ),
          validateError
            ? el('p', { className: 'text-danger text-sm mt-2' },
              typeof validateError === 'string' ? validateError : validateError.message)
            : null,
          validateResult
            ? el('div', { className: 'mt-2', 'data-testid': 'validate-bid-result' },
              validateResult.valid
                ? renderStatusBadge('ok', { label: 'Valid OpenRTB 2.6 request' })
                : renderStatusBadge('error', { label: 'Validation failed' }),
              validateResult.errors?.length
                ? el('ul', { className: 'plain-list mt-2' },
                  validateResult.errors.map((msg) => el('li', { className: 'plain-list__item text-sm' }, msg)),
                )
                : null,
            )
            : null,
        ],
      }),
      renderSectionCard({
        title: 'Shadow diff (1h)',
        icon: 'git-compare',
        desc: 'Shadow vs live winner parity — gate before RTB_MODE=live.',
        children: [renderShadowSummary(shadow)],
      }),
      renderFloorsPanel(),
      renderReconcilePanel(),
    );

    if (!canWrite) {
      children.push(el('p', { className: 'text-muted text-sm' }, 'rtb:write required to create PMP deals.'));
    }

    replaceChildren(container, el('section', { 'data-testid': 'rtb-integration-view' }, ...children));
  }

  load();
  render();

  return {
    destroy() {
      destroyed = true;
    },
  };
}
