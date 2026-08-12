import type { RouteContext, ViewHandle } from '../lib/router_types.js';
import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import * as auth from '../helpers/auth.js';
import { fetchTelegramSummary } from '../helpers/tg_report_api.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderStatTile, renderSubsection } from '../ui/stat_tile.js';
import {
  buildTelegramReportParams,
  createTelegramReportController,
  renderTelegramReportFilters,
  renderTelegramReportHeader,
  renderTelegramReportNav,
  syncTelegramReportUrl,
  telegramFilterHandlers,
} from './tg_report_shell.js';

const PAGE_PATH = '/reports/telegram';

/**
 * Mount Telegram Mini Apps summary report view.
 *
 * @param {HTMLElement} container
 * @param {{ query: URLSearchParams }} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container: HTMLElement, ctx: RouteContext): ViewHandle {
  let destroyed = false;
  const user = auth.getUser();
  const ctrl = createTelegramReportController(ctx, user, PAGE_PATH);
  let loading = false;
  let data: any = null;
  /** @type {Error|null} */
  let error: any = null;
  /** @type {{ destroy: () => void }|null} */
  let funnelChart: any = null;
  let funnelChartMount: any = null;

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

    const [res, err] = await to(fetchTelegramSummary(
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

  function mountFunnelChart() {
    funnelChart?.destroy();
    funnelChart = null;
    if (!funnelChartMount || !data) return;
    const items = [
      { label: 'Clicks', value: data.clicks },
      { label: 'Impressions', value: data.impressions },
      { label: 'Conversions', value: data.conversions },
      { label: 'Premium', value: data.premium },
      { label: 'Motivated', value: data.motivated },
    ];
    import('../charts/pie_chart.js').then((mod) => {
      if (destroyed || !funnelChartMount) return;
      funnelChart = mod.mountPieChart(funnelChartMount, items, 'Telegram attribution funnel');
    });
  }

  function render() {
    if (destroyed) return;
    if (error) {
      funnelChart?.destroy();
      funnelChart = null;
      replaceChildren(container, renderErrorBlock(error));
      return;
    }

    if (loading) {
      funnelChart?.destroy();
      funnelChart = null;
    }

    const funnelItems = data
      ? [
          { label: 'Clicks', val: data.clicks },
          { label: 'Impressions', val: data.impressions },
          { label: 'Conversions', val: data.conversions },
          { label: 'Premium users', val: data.premium },
          { label: 'Motivated clicks', val: data.motivated },
        ]
      : [];

    funnelChartMount = el('div');

    replaceChildren(container,
      renderTelegramReportHeader('Telegram Mini Apps', data?.freshness ?? null),
      renderTelegramReportNav(PAGE_PATH, ctrl.state),
      renderTelegramReportFilters(ctrl.state, telegramFilterHandlers(ctrl, { loading, onSubmit: load, onRerender: render })),
      loading ? el('p', { className: 'loading-hint' }, 'Loading report…') : null,
      !loading && data
        ? el('div', { className: 'stack stack--lg' },
          renderSubsection('Attribution funnel',
            el('div', { className: 'funnel-grid' },
              funnelItems.map((item: any) => renderStatTile(item.label, item.val)),
            ),
          ),
          renderSubsection('Volume breakdown', funnelChartMount),
        )
        : null,
      !loading && !data ? el('p', { className: 'empty-hint' }, 'Perform a query to load report.') : null,
    );

    if (!loading && data) mountFunnelChart();
  }

  ctrl.refreshCampaignOptions().then(() => {
    if (!destroyed) load();
  });

  return {
    destroy() {
      destroyed = true;
      funnelChart?.destroy();
    },
  };
}
