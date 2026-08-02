import { mountReportQuery } from './report_query.js';

/**
 * Mount the placements report view.
 *
 * @param {HTMLElement} container
 * @param {import('../lib/router.js').RouteContext} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container, ctx) {
  return mountReportQuery(container, ctx, {
    endpoint: 'placements',
    title: 'Report: placements',
    rowKey: (row) => `${row.placement_id}-${row.campaign_id}`,
  });
}
