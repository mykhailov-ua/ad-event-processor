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
        el('div', { className: 'page-header' },
          el('h1', { className: 'page-header__title' }, 'Deal performance'),
        ),
        el('p', { className: 'loading-hint' }, 'Loading deals…'),
      );
      return;
    }

    const children = [
      el('div', { className: 'page-header' },
        el('h1', { className: 'page-header__title' }, 'Deal performance'),
        el('p', { className: 'page-header__desc' }, 'PMP deal win rate, bid rate, and fill metrics (skeleton).'),
      ),
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
      el('div', { className: 'table-wrapper table-section' },
        el('table', { className: 'data-table', 'data-testid': 'rtb-deals-table' },
          el('thead', null,
            el('tr', null,
              el('th', { scope: 'col' }, 'Deal ID'),
              el('th', { scope: 'col' }, 'Placement'),
              el('th', { scope: 'col' }, 'Win rate'),
              el('th', { scope: 'col' }, 'Bid rate'),
              el('th', { scope: 'col' }, 'Fill'),
              el('th', { scope: 'col' }, 'Sample N'),
            ),
          ),
          el('tbody', null,
            el('tr', null,
              el('td', { colSpan: 6 }, 'No deal rows — connect RTB deals API.'),
            ),
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
