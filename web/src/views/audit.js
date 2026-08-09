import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { tableSkeletonRows } from '../ui/data_table.js';

const PAGE_SIZE = 50;

/**
 * Mount audit log viewer.
 *
 * @param {HTMLElement} container
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container) {
  let destroyed = false;
  let page = 0;
  let redactPii = true;
  let rows = [];
  let total = 0;
  let loading = true;
  let error = null;

  async function load() {
    loading = true;
    render();
    const params = new URLSearchParams({
      limit: String(PAGE_SIZE),
      offset: String(page * PAGE_SIZE),
      redact_pii: redactPii ? 'true' : 'false',
    });
    const [res, err] = await to(api(`/api/v1/audit?${params.toString()}`));
    if (destroyed) return;
    loading = false;
    if (err) {
      error = err;
      render();
      return;
    }
    error = null;
    rows = Array.isArray(res?.data) ? res.data : [];
    const hdr = res?.headers?.get?.('X-Total-Count');
    total = hdr ? Number(hdr) : rows.length;
    render();
  }

  function render() {
    if (destroyed) return;
    if (error) {
      replaceChildren(container, renderErrorBlock(error, 'Failed to load audit log'));
      return;
    }
    const totalPages = Math.ceil(total / PAGE_SIZE) || 1;
    replaceChildren(container,
      el('div', { className: 'page-header' },
        el('h1', { className: 'page-header__title' }, 'Audit log'),
        el('p', { className: 'text-muted text-sm' }, `${total} entries`),
      ),
      el('label', { className: 'form-check mb-3' },
        el('input', {
          type: 'checkbox',
          checked: redactPii,
          onChange: (e) => {
            redactPii = e.target.checked;
            page = 0;
            load();
          },
        }),
        ' Redact PII in changes/metadata',
      ),
      el('div', { className: 'table-wrapper table-wrapper--scroll elevation-raised' },
        el('table', { className: 'data-table', 'aria-label': 'Audit log' },
          el('thead', null,
            el('tr', null,
              el('th', { scope: 'col' }, 'Time'),
              el('th', { scope: 'col' }, 'Action'),
              el('th', { scope: 'col' }, 'Target'),
              el('th', { scope: 'col' }, 'Admin'),
            ),
          ),
          el('tbody', null,
            loading && rows.length === 0 ? tableSkeletonRows(4) : null,
            !loading && rows.length === 0
              ? el('tr', null, el('td', { colSpan: 4 }, 'No audit entries.'))
              : null,
            rows.map((row) => el('tr', null,
              el('td', null, row.created_at ? new Date(row.created_at).toLocaleString() : '—'),
              el('td', null, row.action ?? '—'),
              el('td', null,
                row.target_type ?? '—',
                row.target_id ? ` · ${row.target_id.slice(0, 8)}…` : '',
              ),
              el('td', { className: 'font-mono text-hint' }, row.admin_id?.slice(0, 8) ?? '—'),
            )),
          ),
        ),
      ),
      totalPages > 1
        ? el('div', { className: 'pagination-bar' },
          el('button', {
            type: 'button',
            className: 'btn btn--secondary btn--sm',
            disabled: page === 0,
            onClick: () => { page -= 1; load(); },
          }, 'Prev'),
          el('span', { className: 'text-muted text-xs' }, `${page + 1} / ${totalPages}`),
          el('button', {
            type: 'button',
            className: 'btn btn--secondary btn--sm',
            disabled: page >= totalPages - 1,
            onClick: () => { page += 1; load(); },
          }, 'Next'),
        )
        : null,
    );
  }

  load();
  return {
    destroy() {
      destroyed = true;
    },
  };
}
