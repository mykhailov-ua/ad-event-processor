import type { ViewHandle } from '../lib/router_types.js';
import type { ReconRunDTO } from '../types/api/ops_extra.js';
import { el, replaceChildren, eventTargetValue } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderStatusBadge } from '../ui/status_badge.js';
import { tableSkeletonRows, renderEmptyTableCell, renderPaginationBar } from '../ui/data_table.js';
import { mountFilterToolbar } from '../ui/filter_toolbar.js';
import { fetchReconRuns } from '../helpers/ops_recon_api.js';

const PAGE_SIZE = 50;

type ServiceFilter = 'all' | 'management' | 'payment';

/**
 * Mount reconciliation runs list for operators.
 */
export function mount(container: HTMLElement): ViewHandle {
  let destroyed = false;
  let page = 0;
  let service: ServiceFilter = 'all';
  let rows: ReconRunDTO[] = [];
  let total = 0;
  let loading = true;
  let error: unknown = null;

  async function load(): Promise<void> {
    loading = true;
    render();
    const [res, err] = await to(fetchReconRuns(service, PAGE_SIZE, page * PAGE_SIZE));
    if (destroyed) return;
    loading = false;
    if (err) {
      error = err;
      rows = [];
      total = 0;
      render();
      return;
    }
    error = null;
    rows = res?.items ?? [];
    total = res?.total ?? rows.length;
    render();
  }

  function render(): void {
    if (destroyed) return;
    if (error) {
      replaceChildren(container, renderErrorBlock(error, 'Failed to load reconciliation runs'));
      return;
    }

    const totalPages = Math.ceil(total / PAGE_SIZE) || 1;
    replaceChildren(container,
      el('section', { 'data-testid': 'ops-recon-view' },
        el('div', { className: 'page-header' },
          el('h1', { className: 'page-header__title' }, 'Reconciliation runs'),
          el('p', { className: 'text-muted text-sm' },
            'Management ledger vs spend checks and payment intent reconciliation.',
          ),
        ),
        el('label', { className: 'form-field mb-3', htmlFor: 'recon-service' },
          'Service',
          el('select', {
            id: 'recon-service',
            className: 'form-input form-input--sm',
            defaultValue: service,
            onChange: (e: Event) => {
              service = eventTargetValue(e) as ServiceFilter;
              page = 0;
              load();
            },
          },
            el('option', { value: 'all' }, 'All'),
            el('option', { value: 'management' }, 'Management'),
            el('option', { value: 'payment' }, 'Payment'),
          ),
        ),
        (() => {
          const bar = el('div', { className: 'mb-4' });
          mountFilterToolbar(bar, {
            pagination: totalPages > 1
              ? renderPaginationBar({
                label: `${page + 1} / ${totalPages}`,
                prevDisabled: page === 0,
                nextDisabled: page >= totalPages - 1,
                onPrev: () => { page -= 1; load(); },
                onNext: () => { page += 1; load(); },
              })
              : null,
          });
          return bar;
        })(),
        el('div', { className: 'table-wrapper elevation-raised' },
          el('table', { className: 'data-table', 'data-testid': 'ops-recon-table' },
            el('thead', null,
              el('tr', null,
                el('th', { scope: 'col' }, 'Service'),
                el('th', { scope: 'col' }, 'Period'),
                el('th', { scope: 'col' }, 'Status'),
                el('th', { scope: 'col' }, 'Discrepancies'),
                el('th', { scope: 'col' }, 'Created'),
              ),
            ),
            el('tbody', null,
              loading && rows.length === 0 ? tableSkeletonRows(5) : null,
              !loading && rows.length === 0
                ? el('tr', null,
                  renderEmptyTableCell(5, {
                    title: 'No reconciliation runs',
                    description: 'Scheduled recon jobs will appear here after they complete.',
                  }),
                )
                : null,
              rows.map((row) => el('tr', null,
                el('td', null, row.service),
                el('td', { className: 'font-mono text-xs' },
                  `${formatPeriod(row.period_start)} → ${formatPeriod(row.period_end)}`,
                ),
                el('td', null, renderStatusBadge(statusTone(row.status), { label: row.status })),
                el('td', null, String(row.discrepancies_found ?? row.findings_count ?? '—')),
                el('td', { className: 'text-muted text-xs' },
                  row.created_at ? new Date(row.created_at).toLocaleString() : '—',
                ),
              )),
            ),
          ),
        ),
      ),
    );
  }

  load();

  return {
    destroy() {
      destroyed = true;
    },
  };
}

function formatPeriod(value: string | undefined): string {
  if (!value) return '—';
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? value : d.toISOString().slice(0, 16).replace('T', ' ');
}

function statusTone(status: string): 'ok' | 'warn' | 'error' | 'neutral' {
  const s = status.toUpperCase();
  if (s === 'COMPLETED' || s === 'OK') return 'ok';
  if (s === 'FAILED' || s === 'ERROR') return 'error';
  if (s === 'RUNNING' || s === 'PENDING') return 'warn';
  return 'neutral';
}
