import { mountReportQuery } from './report_query.js';

/**
 * @param {HTMLElement} container
 * @param {import('../lib/router.js').RouteContext} ctx
 */
export function mount(container, ctx) {
  return mountReportQuery(container, ctx, {
    endpoint: 'placements',
    title: 'Report: placements',
    rowKey: (row) => `${row.placement_id}-${row.campaign_id}`,
  });
}
