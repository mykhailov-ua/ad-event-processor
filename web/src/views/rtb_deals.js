import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import { probeStart, probeEnd } from '../helpers/perf_probe.js';
import { renderPerfBlock } from '../helpers/perf_display.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderStubBanner } from '../ui/stub_banner.js';

/**
 * Mount RTB deal performance skeleton for buyer/agency workflows.
 *
 * @param {HTMLElement} container
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container) {
  let destroyed = false;
  const state = { loading: true, error: null, deals: [], stub: false };

  async function load() {
    const probe = probeStart('rtb.deals.list');
    const [, err] = await to(api('/api/v1/rtb/deals'));
    probeEnd(probe, { allocs: 1, bytes: 128 });
    if (destroyed) return;
    if (err) {
      state.error = err;
      state.stub = err.status === 404 || err.status === 501;
      state.loading = false;
      render();
      return;
    }
    state.loading = false;
    render();
  }

  function render() {
    if (destroyed) return;

    if (state.loading) {
      replaceChildren(container,
        el('section', null,
          el('h1', null, 'Deal performance'),
          el('p', null, 'Loading deals…'),
        ),
      );
      return;
    }

    const children = [
      el('h1', null, 'Deal performance'),
      el('p', null, 'PMP deal win rate, bid rate, and fill metrics (skeleton).'),
    ];

    if (state.error) {
      children.push(
        state.stub
          ? renderStubBanner({
            message: 'Deals API is not fully available yet. Skeleton shows expected columns.',
          })
          : renderErrorBlock(state.error, 'Failed to load deals'),
      );
    }

    children.push(
      el('table', { 'data-testid': 'rtb-deals-table' },
        el('thead', null,
          el('tr', null,
            el('th', null, 'Deal ID'),
            el('th', null, 'Placement'),
            el('th', null, 'Win rate'),
            el('th', null, 'Bid rate'),
            el('th', null, 'Fill'),
            el('th', null, 'Sample N'),
          ),
        ),
        el('tbody', null,
          el('tr', null,
            el('td', { colSpan: 6 }, 'No deal rows — connect RTB deals API.'),
          ),
        ),
      ),
      renderPerfBlock('rtb-deals-perf'),
    );

    replaceChildren(container, el('section', { 'data-testid': 'rtb-deals-view' }, ...children));
  }

  load();
  render();

  return {
    destroy() {
      destroyed = true;
    },
  };
}
