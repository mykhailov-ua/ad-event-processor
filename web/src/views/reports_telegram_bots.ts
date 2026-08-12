import type { RouteContext, ViewHandle } from '../lib/router_types.js';
import type { DataFreshness, ReportEnvelope, ReportRow } from '../types/api/report.js';
import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import * as auth from '../helpers/auth.js';
import { fetchTelegramBots } from '../helpers/tg_report_api.js';
import { renderErrorBlock } from '../ui/error_block.js';
import {
  buildTelegramReportParams,
  createTelegramReportController,
  renderTelegramReportEmpty,
  renderTelegramReportFilters,
  renderTelegramReportHeader,
  renderTelegramReportNav,
  syncTelegramReportUrl,
  telegramFilterHandlers,
} from './tg_report_shell.js';

const PAGE_PATH = '/reports/telegram/bots';

export function mount(container: HTMLElement, ctx: RouteContext): ViewHandle {
  let destroyed = false;
  const user = auth.getUser();
  const ctrl = createTelegramReportController(ctx, user, PAGE_PATH);
  let loading = false;
  /** @type {ReportRow[]|null} */
  let rows: ReportRow[] | null = null;
  /** @type {DataFreshness|null} */
  let freshness: DataFreshness | null = null;
  /** @type {Error|null} */
  let error: any = null;

  async function load() {
    const validationErr = ctrl.validateBeforeLoad();
    if (validationErr) {
      error = new Error(validationErr);
      render();
      return;
    }
    loading = true;
    error = null;
    render();

    const [res, err] = await to(fetchTelegramBots(
      buildTelegramReportParams(ctrl.state, ctrl.sessionScoped, user),
    ));
    if (destroyed) return;
    loading = false;
    if (err) {
      error = err;
      rows = null;
      freshness = null;
    } else {
      const payload = (res as ReportEnvelope | null) ?? null;
      rows = payload?.rows ?? [];
      freshness = payload?.freshness ?? null;
      syncTelegramReportUrl(PAGE_PATH, ctrl.state);
    }
    render();
  }

  function render() {
    if (destroyed) return;
    if (error) {
      replaceChildren(container, renderErrorBlock(error));
      return;
    }

    const table = rows && rows.length > 0
      ? el('div', { className: 'table-wrapper table-section' },
        el('table', { className: 'data-table' },
          el('thead', null,
            el('tr', null,
              el('th', { scope: 'col' }, 'Bot ID'),
              el('th', { scope: 'col' }, 'Clicks'),
              el('th', { scope: 'col' }, 'Impressions'),
              el('th', { scope: 'col' }, 'Premium'),
            ),
          ),
          el('tbody', null,
            rows.map((row: any) => el('tr', null,
              el('td', null, String(row.bot_id ?? '—')),
              el('td', null, String(row.clicks ?? 0)),
              el('td', null, String(row.impressions ?? 0)),
              el('td', null, String(row.premium ?? 0)),
            )),
          ),
        ),
      )
      : (!loading ? renderTelegramReportEmpty() : null);

    replaceChildren(container,
      renderTelegramReportHeader('Telegram Bots', freshness),
      renderTelegramReportNav(PAGE_PATH, ctrl.state),
      renderTelegramReportFilters(ctrl.state, telegramFilterHandlers(ctrl, { loading, onSubmit: load, onRerender: render })),
      loading ? el('p', { className: 'loading-hint' }, 'Loading…') : null,
      !loading ? table : null,
    );
  }

  ctrl.refreshCampaignOptions().then(() => {
    if (!destroyed) load();
  });

  return { destroy() { destroyed = true; } };
}
