import { mountReportQuery } from './report_query.js';

/**
 * @param {HTMLElement} container
 * @param {import('../lib/router.js').RouteContext} ctx
 */
export function mount(container, ctx) {
  return mountReportQuery(container, ctx, {
    endpoint: 'keywords',
    title: 'Report: keywords',
    rowKey: (row) => `${row.keyword}-${row.campaign_id}`,
  });
}
