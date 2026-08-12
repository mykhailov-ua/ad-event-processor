import type { ViewHandle } from '../lib/router_types.js';
import type { AuditLogRow } from '../types/api/index.js';
import { el, replaceChildren, eventTargetChecked } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import { apiBlobResult } from '../helpers/api_blob.js';
import { can } from '../helpers/permissions.js';
import * as auth from '../helpers/auth.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { tableSkeletonRows, renderEmptyTableCell, renderPaginationBar } from '../ui/data_table.js';
import { renderButton } from '../ui/button.js';
import { mountFilterToolbar } from '../ui/filter_toolbar.js';

const PAGE_SIZE = 50;

/**
 * Mount audit log viewer.
 *
 * @param {HTMLElement} container
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container: HTMLElement): ViewHandle {
  let destroyed = false;
  let page = 0;
  let redactPii = true;
  let rows: AuditLogRow[] = [];
  let total = 0;
  let loading = true;
  let error: unknown = null;
  let exportBusy = false;
  let exportTruncated = false;
  let exportNextCursor: string | null = null;

  const user = auth.getUser();
  const canExport = can(user?.permissions ?? [], 'audit:read');

  function downloadBlob(blob: Blob, filename: string): void {
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = filename;
    anchor.click();
    URL.revokeObjectURL(url);
  }

  async function exportCsv(): Promise<void> {
    if (!canExport || exportBusy) return;
    exportBusy = true;
    exportTruncated = false;
    exportNextCursor = null;
    render();

    const params = new URLSearchParams({
      format: 'csv',
      redact_pii: redactPii ? 'true' : 'false',
    });
    const [result, err] = await to(apiBlobResult(`/api/v1/audit/export?${params.toString()}`));
    exportBusy = false;
    if (destroyed) return;
    if (err) {
      const view = mapServiceError(err);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      render();
      return;
    }
    exportTruncated = result.truncated;
    exportNextCursor = result.nextCursor;
    downloadBlob(result.blob, 'audit-export.csv');
    render();
  }

  async function load() {
    loading = true;
    render();
    const params = new URLSearchParams({
      limit: String(PAGE_SIZE),
      offset: String(page * PAGE_SIZE),
      redact_pii: redactPii ? 'true' : 'false',
    });
    const [res, err] = await to(api<AuditLogRow[]>(`/api/v1/audit?${params.toString()}`));
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
        el('div', { className: 'page-header__row' },
          el('h1', { className: 'page-header__title' }, 'Audit log'),
          canExport
            ? renderButton({
              label: 'Export CSV',
              variant: 'secondary',
              size: 'sm',
              className: 'ml-auto',
              loading: exportBusy,
              testId: 'audit-export-csv',
              onClick: exportCsv,
            })
            : null,
        ),
        el('p', { className: 'text-muted text-sm' }, `${total} entries`),
      ),
      exportTruncated
        ? el('p', {
          className: 'text-warning text-sm mb-2',
          'data-testid': 'audit-export-truncated',
        }, `Last export was truncated at cursor ${exportNextCursor ?? '—'}.`)
        : null,
      el('label', { className: 'form-check mb-3' },
        el('input', {
          type: 'checkbox',
          checked: redactPii,
          onChange: (e: Event) => {
            redactPii = eventTargetChecked(e);
            page = 0;
            load();
          },
        }),
        ' Redact PII in changes/metadata',
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
              ? el('tr', null,
                renderEmptyTableCell(4, {
                  title: 'No audit entries',
                  description: 'Admin actions will appear here as they occur.',
                }),
              )
              : null,
            rows.map((row: AuditLogRow) => el('tr', null,
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
    );
  }

  load();
  return {
    destroy() {
      destroyed = true;
    },
  };
}
