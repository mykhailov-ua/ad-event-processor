import type { RouteContext, ViewHandle } from '../lib/router_types.js';
import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import * as auth from '../helpers/auth.js';
import { fetchTelegramPremium } from '../helpers/tg_report_api.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderStatTile, renderStatsRow } from '../ui/stat_tile.js';
import {
  buildTelegramReportParams,
  createTelegramReportController,
  renderTelegramReportFilters,
  renderTelegramReportHeader,
  renderTelegramReportNav,
  syncTelegramReportUrl,
  telegramFilterHandlers,
} from './tg_report_shell.js';

const PAGE_PATH = '/reports/telegram/premium';

export function mount(container: HTMLElement, ctx: RouteContext): ViewHandle {
  let destroyed = false;
  const user = auth.getUser();
  const ctrl = createTelegramReportController(ctx, user, PAGE_PATH);
  let loading = false;
  let data: any = null;
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

    const [res, err] = await to(fetchTelegramPremium(
      buildTelegramReportParams(ctrl.state, ctrl.sessionScoped, user),
    ));
    if (destroyed) return;
    loading = false;
    if (err) {
      error = err;
      data = null;
    } else {
      data = res;
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

    const stats = data
      ? renderStatsRow(
        renderStatTile('Premium clicks', data.premium_clicks ?? 0),
        renderStatTile('Non-premium clicks', data.non_premium_clicks ?? 0),
        renderStatTile('Premium rate', `${(data.premium_rate_pct ?? 0).toFixed(1)}%`),
      )
      : null;

    replaceChildren(container,
      renderTelegramReportHeader('Telegram Premium', data?.freshness ?? null),
      renderTelegramReportNav(PAGE_PATH, ctrl.state),
      renderTelegramReportFilters(ctrl.state, telegramFilterHandlers(ctrl, { loading, onSubmit: load, onRerender: render })),
      loading ? el('p', { className: 'loading-hint' }, 'Loading…') : null,
      !loading && stats ? stats : null,
      !loading && !data ? el('p', { className: 'empty-hint' }, 'Perform a query to load report.') : null,
    );
  }

  ctrl.refreshCampaignOptions().then(() => {
    if (!destroyed) load();
  });

  return { destroy() { destroyed = true; } };
}
