import { el } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { renderFormField } from '../ui/form_field.js';
import { renderFreshnessBadge } from '../ui/freshness_badge.js';
import { REPORT_DATE_PRESETS } from '../helpers/date_presets.js';
import { buyerEmptyCopy } from '../models/empty_state.js';
import { boundCustomerId, hasBoundCustomer } from '../helpers/buyer_session.js';
import { validateCustomerIdField, validateReportRange } from '../helpers/validators.js';
import { tenantReportQueryString } from '../helpers/tenant_url.js';
import { fetchCampaignOptions } from '../helpers/campaign_picker.js';
import {
  downloadReportExport,
  pollReportJob,
  submitReportExport,
} from '../helpers/report_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { createInFlightGuard } from '../lib/async_guard.js';
import { telegramPiiNotice } from '../helpers/tg_pii.js';

/**
 * @typedef {{
 *   customerInput: string,
 *   campaignInput: string,
 *   from: string,
 *   to: string,
 *   activePreset: string,
 * }} TelegramReportState
 */

/** Nav entries for Telegram analytics sub-pages. */
export const TELEGRAM_REPORT_PAGES = [
  { path: '/reports/telegram', label: 'Summary' },
  { path: '/reports/telegram/funnel', label: 'Funnel' },
  { path: '/reports/telegram/bots', label: 'Bots' },
  { path: '/reports/telegram/premium', label: 'Premium' },
  { path: '/reports/telegram/fraud', label: 'Fraud' },
];

/**
 * Initialize filter state from route query string.
 *
 * @param {URLSearchParams} query
 * @param {{ sessionScoped?: boolean, user?: { customer_id?: string }|null }} [opts]
 * @returns {TelegramReportState}
 */
export function createTelegramReportState(query, opts = {}) {
  const preset = REPORT_DATE_PRESETS[1] ?? REPORT_DATE_PRESETS[0];
  const sessionScoped = Boolean(opts.sessionScoped);
  const user = opts.user;
  return {
    customerInput: query.get('customer_id')
      || (sessionScoped && user ? boundCustomerId(user) : ''),
    campaignInput: query.get('campaign_id') || '',
    from: query.get('from') || preset.from(),
    to: query.get('to') || preset.to(),
    activePreset: query.get('preset') || preset.id,
  };
}

/**
 * Resolve effective customer_id for API calls.
 *
 * @param {TelegramReportState} state
 * @param {boolean} sessionScoped
 * @param {{ customer_id?: string }|null|undefined} user
 * @returns {string}
 */
export function resolveTelegramCustomerId(state, sessionScoped, user) {
  return sessionScoped ? boundCustomerId(user) : state.customerInput.trim();
}

/**
 * Build API query params from report state.
 *
 * @param {TelegramReportState} state
 * @param {boolean} sessionScoped
 * @param {{ customer_id?: string }|null|undefined} user
 * @returns {{ from: string, to: string, customerId?: string, campaignId?: string }}
 */
export function buildTelegramReportParams(state, sessionScoped, user) {
  const params = { from: state.from, to: state.to };
  const customerId = resolveTelegramCustomerId(state, sessionScoped, user);
  if (customerId) params.customerId = customerId;
  const campaignId = state.campaignInput.trim();
  if (campaignId) params.campaignId = campaignId;
  return params;
}

/**
 * Apply a date preset to report state.
 *
 * @param {TelegramReportState} state
 * @param {{ id: string, from: () => string, to: () => string }} preset
 */
export function applyTelegramPreset(state, preset) {
  state.activePreset = preset.id;
  state.from = preset.from();
  state.to = preset.to();
}

/**
 * Sync report filters to the browser URL (shareable links).
 *
 * @param {string} path
 * @param {TelegramReportState} state
 */
export function syncTelegramReportUrl(path, state) {
  try {
    const qs = tenantReportQueryString({
      customer_id: state.customerInput.trim(),
      campaign_id: state.campaignInput.trim(),
      from: state.from,
      to: state.to,
      preset: state.activePreset,
    });
    window.history.replaceState(null, '', qs ? `${path}?${qs}` : path);
  } catch {
    // ignore
  }
}

/**
 * Build href for a Telegram report sub-page preserving current filters.
 *
 * @param {string} path
 * @param {TelegramReportState} state
 * @returns {string}
 */
export function telegramReportHref(path, state) {
  const qs = tenantReportQueryString({
    customer_id: state.customerInput.trim(),
    campaign_id: state.campaignInput.trim(),
    from: state.from,
    to: state.to,
    preset: state.activePreset,
  });
  return qs ? `${path}?${qs}` : path;
}

/**
 * Create shared controller state for Telegram report pages.
 *
 * @param {{ query: URLSearchParams }} ctx
 * @param {{ customer_id?: string, role?: string }|null} user
 * @param {string} pagePath
 * @returns {{
 *   state: TelegramReportState,
 *   sessionScoped: boolean,
 *   campaignOptions: Array<{ id: string, name: string }>,
 *   exportStatus: string|null,
 *   exportLoading: boolean,
 *   validateBeforeLoad: () => string|null,
 *   refreshCampaignOptions: () => Promise<void>,
 *   handleExport: () => Promise<void>,
 * }}
 */
export function createTelegramReportController(ctx, user, pagePath) {
  const sessionScoped = hasBoundCustomer(user?.role);
  const state = createTelegramReportState(ctx.query, { sessionScoped, user });
  /** @type {Array<{ id: string, name: string }>} */
  let campaignOptions = [];
  let exportStatus = null;
  let exportLoading = false;
  const exportGate = createInFlightGuard();

  function validateBeforeLoad() {
    const customerId = resolveTelegramCustomerId(state, sessionScoped, user);
    if (!sessionScoped) {
      const customerErr = validateCustomerIdField(state.customerInput);
      if (customerErr) return customerErr;
    } else if (!customerId) {
      return 'customer_id required';
    }
    return validateReportRange(state.from, state.to);
  }

  async function refreshCampaignOptions() {
    const customerId = resolveTelegramCustomerId(state, sessionScoped, user);
    if (!customerId) {
      campaignOptions = [];
      return;
    }
    const [opts] = await to(fetchCampaignOptions(customerId));
    campaignOptions = opts ?? [];
    if (state.campaignInput && !campaignOptions.some((c) => c.id === state.campaignInput)) {
      campaignOptions = [{ id: state.campaignInput, name: state.campaignInput }, ...campaignOptions];
    }
  }

  async function handleExport() {
    const customerId = resolveTelegramCustomerId(state, sessionScoped, user);
    const rangeErr = validateReportRange(state.from, state.to);
    const customerErr = sessionScoped ? null : validateCustomerIdField(state.customerInput);
    if (!customerId || rangeErr || customerErr) {
      pushToastMessage({
        title: 'Export blocked',
        message: rangeErr || customerErr || 'customer_id required',
      });
      return;
    }
    if (!exportGate.tryAcquire()) return;
    exportLoading = true;
    exportStatus = null;
    const result = await submitReportExport({
      customerId,
      reportKey: 'telegram',
      from: state.from,
      to: state.to,
    });
    if (!result.ok || !result.jobId) {
      exportLoading = false;
      exportGate.release();
      exportStatus = result.message;
      pushToastMessage({ title: 'Export failed', message: result.message });
      return;
    }
    const polled = await pollReportJob(result.jobId);
    if (polled.ok) {
      await downloadReportExport(result.jobId, 'telegram-report.csv');
      exportStatus = 'Export downloaded';
      pushToastMessage({ title: 'Export ready', message: 'telegram-report.csv downloaded' });
    } else {
      exportStatus = polled.message;
      pushToastMessage({ title: 'Export failed', message: polled.message });
    }
    exportLoading = false;
    exportGate.release();
    syncTelegramReportUrl(pagePath, state);
  }

  return {
    get state() { return state; },
    sessionScoped,
    get campaignOptions() { return campaignOptions; },
    get exportStatus() { return exportStatus; },
    get exportLoading() { return exportLoading; },
    validateBeforeLoad,
    refreshCampaignOptions,
    handleExport,
  };
}

/**
 * Render horizontal nav between Telegram report pages.
 *
 * @param {string} activePath
 * @param {TelegramReportState} state
 * @returns {HTMLElement}
 */
export function renderTelegramReportNav(activePath, state) {
  return el('nav', { className: 'tab-bar', 'aria-label': 'Telegram reports' },
    TELEGRAM_REPORT_PAGES.map((page) =>
      el('a', {
        href: telegramReportHref(page.path, state),
        className: 'tab-bar__item' + (activePath === page.path ? ' tab-bar__item--active' : ''),
        'aria-current': activePath === page.path ? 'page' : undefined,
      }, page.label),
    ),
  );
}

/**
 * Render date presets row.
 *
 * @param {TelegramReportState} state
 * @param {(preset: typeof REPORT_DATE_PRESETS[number]) => void} onPreset
 * @returns {HTMLElement}
 */
export function renderTelegramDatePresets(state, onPreset) {
  return el('div', { className: 'date-presets' },
    el('span', { className: 'date-presets__label' }, 'Range'),
    REPORT_DATE_PRESETS.map((preset) =>
      el('button', {
        type: 'button',
        className: 'date-preset' + (state.activePreset === preset.id ? ' date-preset--active' : ''),
        onClick: () => onPreset(preset),
      }, preset.label),
    ),
  );
}

/**
 * Render shared filter form for Telegram reports.
 *
 * @param {TelegramReportState} state
 * @param {{
 *   sessionScoped: boolean,
 *   loading: boolean,
 *   campaignOptions: Array<{ id: string, name: string }>,
 *   exportLoading?: boolean,
 *   exportStatus?: string|null,
 *   onSubmit: () => void,
 *   onCustomerInput?: (v: string) => void,
 *   onCampaignInput: (v: string) => void,
 *   onFromInput: (v: string) => void,
 *   onToInput: (v: string) => void,
 *   onPreset: (preset: typeof REPORT_DATE_PRESETS[number]) => void,
 *   onExport?: () => void,
 * }} handlers
 * @returns {HTMLElement}
 */
export function renderTelegramReportFilters(state, handlers) {
  const campaignField = handlers.campaignOptions.length > 0
    ? el('select', {
        id: 'tg-report-campaign',
        className: 'form-input',
        value: state.campaignInput,
        onChange: (e) => handlers.onCampaignInput(e.target.value),
      },
        el('option', { value: '' }, 'All campaigns'),
        handlers.campaignOptions.map((c) =>
          el('option', { value: c.id, selected: state.campaignInput === c.id }, c.name),
        ),
      )
    : el('input', {
        id: 'tg-report-campaign',
        className: 'form-input',
        value: state.campaignInput,
        placeholder: 'All campaigns (UUID)',
        onInput: (e) => handlers.onCampaignInput(e.target.value),
      });

  return el('form', {
    className: 'filter-form settings-panel',
    onSubmit: (e) => {
      e.preventDefault();
      handlers.onSubmit();
    },
  },
    !handlers.sessionScoped
      ? renderFormField({
        label: 'Customer ID',
        htmlFor: 'tg-report-customer',
        children: el('input', {
          id: 'tg-report-customer',
          className: 'form-input',
          value: state.customerInput,
          placeholder: 'customer_id (UUID)',
          onInput: (e) => handlers.onCustomerInput?.(e.target.value),
        }),
      })
      : null,
    renderFormField({
      label: 'Campaign',
      htmlFor: 'tg-report-campaign',
      children: campaignField,
    }),
    renderTelegramDatePresets(state, handlers.onPreset),
    renderFormField({
      label: 'From',
      htmlFor: 'tg-report-from',
      children: el('input', {
        id: 'tg-report-from',
        className: 'form-input',
        value: state.from,
        onInput: (e) => handlers.onFromInput(e.target.value),
      }),
    }),
    renderFormField({
      label: 'To',
      htmlFor: 'tg-report-to',
      children: el('input', {
        id: 'tg-report-to',
        className: 'form-input',
        value: state.to,
        onInput: (e) => handlers.onToInput(e.target.value),
      }),
    }),
    el('div', { className: 'toolbar-row' },
      el('button', { type: 'submit', className: 'btn btn--primary', disabled: handlers.loading }, 'Query'),
      handlers.onExport
        ? el('button', {
          type: 'button',
          className: 'btn btn--secondary',
          disabled: handlers.loading || handlers.exportLoading,
          onClick: () => handlers.onExport?.(),
        }, handlers.exportLoading ? 'Exporting…' : 'Export CSV')
        : null,
      handlers.exportStatus
        ? el('span', { className: 'text-muted text-sm' }, handlers.exportStatus)
        : null,
    ),
  );
}

/**
 * Render page header with optional freshness badge.
 *
 * @param {string} title
 * @param {{ stale?: boolean, lag_seconds?: number }|null} freshness
 * @returns {HTMLElement}
 */
export function renderTelegramReportHeader(title, freshness) {
  return el('div', { className: 'page-header' },
    el('h1', { className: 'page-header__title' }, title),
    freshness
      ? renderFreshnessBadge({ stale: freshness.stale, lagSeconds: freshness.lag_seconds })
      : null,
    el('p', { className: 'page-header__desc text-sm' }, telegramPiiNotice()),
  );
}

/**
 * Render empty-state block when a report has no rows.
 *
 * @returns {HTMLElement}
 */
export function renderTelegramReportEmpty() {
  const copy = buyerEmptyCopy('reports_empty');
  return el('div', { className: 'empty-state section-block' },
    el('h2', { className: 'empty-state__title' }, copy.title),
    el('p', null, copy.description),
    copy.actionHref
      ? el('a', { href: copy.actionHref, className: 'btn btn--secondary' }, copy.actionLabel ?? 'Open')
      : null,
  );
}

/**
 * Build standard filter handlers for a Telegram report page.
 *
 * @param {ReturnType<typeof createTelegramReportController>} ctrl
 * @param {{ loading: boolean, onSubmit: () => void, onRerender: () => void }} opts
 */
export function telegramFilterHandlers(ctrl, opts) {
  const state = ctrl.state;
  return {
    sessionScoped: ctrl.sessionScoped,
    loading: opts.loading,
    campaignOptions: ctrl.campaignOptions,
    exportLoading: ctrl.exportLoading,
    exportStatus: ctrl.exportStatus,
    onSubmit: opts.onSubmit,
    onCustomerInput: (v) => {
      state.customerInput = v;
      state.campaignInput = '';
      ctrl.refreshCampaignOptions().then(opts.onRerender);
    },
    onCampaignInput: (v) => { state.campaignInput = v; },
    onFromInput: (v) => { state.from = v; state.activePreset = ''; },
    onToInput: (v) => { state.to = v; state.activePreset = ''; },
    onPreset: (preset) => {
      applyTelegramPreset(state, preset);
      opts.onSubmit();
    },
    onExport: async () => {
      await ctrl.handleExport();
      opts.onRerender();
    },
  };
}
