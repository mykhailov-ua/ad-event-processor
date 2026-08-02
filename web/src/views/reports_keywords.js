import { mountReportQuery } from './report_query.js';

/**
 * Mount the keywords report view.
 *
 * @param {HTMLElement} container
 * @param {import('../lib/router.js').RouteContext} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container, ctx) {
  return mountReportQuery(container, ctx, {
    endpoint: 'keywords',
    title: 'Report: keywords',
    rowKey: (row) => `${row.keyword}-${row.campaign_id}`,
  });
}
