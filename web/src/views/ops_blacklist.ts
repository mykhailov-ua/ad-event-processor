import type { ViewHandle } from '../lib/router_types.js';
import type { BlacklistEntryDTO, BlacklistListResponse } from '../types/api/index.js';
import { el, replaceChildren, eventTargetValue } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import { apiConfirmed } from '../helpers/confirmed_api.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { tableSkeletonRows, renderEmptyTableCell, renderPaginationBar } from '../ui/data_table.js';
import { renderButton } from '../ui/button.js';
import { mountFilterToolbar } from '../ui/filter_toolbar.js';

const PAGE_SIZE = 50;

type BlacklistForm = {
  ip: string;
  source: string;
  ttl: string;
};

/**
 * Mount ops IP blacklist management UI.
 *
 * @param {HTMLElement} container
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container: HTMLElement): ViewHandle {
  let destroyed = false;
  let page = 0;
  let items: BlacklistEntryDTO[] = [];
  let total = 0;
  let loading = true;
  let error: unknown = null;
  let preview: unknown = null;
  const form: BlacklistForm = { ip: '', source: 'manual', ttl: '' };
  let busy = false;

  async function load() {
    loading = true;
    render();
    const offset = page * PAGE_SIZE;
    const [res, err] = await to(api<BlacklistListResponse>(`/api/v1/ops/blacklist?limit=${PAGE_SIZE}&offset=${offset}`));
    if (destroyed) return;
    loading = false;
    if (err) {
      error = err;
      render();
      return;
    }
    error = null;
    const data = res?.data ?? {};
    items = data.items ?? [];
    total = data.total ?? items.length;
    render();
  }

  async function dryRun() {
    preview = null;
    const [res, err] = await to(api('/api/v1/ops/blacklist', {
      method: 'POST',
      headers: { 'X-Dry-Run': '1' },
      body: JSON.stringify({
        ip: form.ip.trim(),
        source: form.source.trim() || 'manual',
        ttl_seconds: form.ttl ? Number(form.ttl) : undefined,
      }),
    }));
    if (destroyed) return;
    if (err) {
      pushToastMessage({ title: 'Preview failed', message: mapServiceError(err).message });
      return;
    }
    preview = res?.data;
    render();
  }

  async function block() {
    busy = true;
    render();
    const [, err] = await to(apiConfirmed('/api/v1/ops/blacklist', {
      method: 'POST',
      body: JSON.stringify({
        ip: form.ip.trim(),
        source: form.source.trim() || 'manual',
        ttl_seconds: form.ttl ? Number(form.ttl) : undefined,
      }),
    }));
    busy = false;
    if (err) {
      if (err instanceof ConfirmCancelledError) {
        render();
        return;
      }
      pushToastMessage({ title: 'Block failed', message: mapServiceError(err).message });
      render();
      return;
    }
    pushToastMessage({ title: 'IP blocked', message: form.ip });
    form.ip = '';
    preview = null;
    load();
  }

  async function unblock(ip: string | undefined, source: string | undefined) {
    const [, err] = await to(apiConfirmed('/api/v1/ops/blacklist', {
      method: 'DELETE',
      body: JSON.stringify({ ip, source }),
    }));
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Unblock failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({ title: 'IP unblocked', message: ip ?? '' });
    load();
  }

  function render() {
    if (destroyed) return;
    if (error) {
      replaceChildren(container, renderErrorBlock(error, 'Failed to load blacklist'));
      return;
    }
    const totalPages = Math.ceil(total / PAGE_SIZE) || 1;
    replaceChildren(container,
      el('div', { className: 'page-header' },
        el('h1', { className: 'page-header__title' }, 'Blacklist'),
        el('p', { className: 'text-muted text-sm' }, 'Edge + manual IP blocks'),
      ),
      el('div', { className: 'section-card stack mb-4' },
        el('h2', { className: 'subsection-title' }, 'Add block'),
        el('label', { className: 'form-field', htmlFor: 'bl-ip' },
          'IP address',
          el('input', {
            id: 'bl-ip',
            className: 'form-input',
            placeholder: '203.0.113.10',
            value: form.ip,
            onInput: (e: Event) => { form.ip = eventTargetValue(e); },
          }),
        ),
        el('label', { className: 'form-field', htmlFor: 'bl-source' },
          'Source',
          el('input', {
            id: 'bl-source',
            className: 'form-input form-input--sm',
            value: form.source,
            onInput: (e: Event) => { form.source = eventTargetValue(e); },
          }),
        ),
        el('label', { className: 'form-field', htmlFor: 'bl-ttl' },
          'TTL seconds (optional)',
          el('input', {
            id: 'bl-ttl',
            className: 'form-input form-input--sm',
            inputMode: 'numeric',
            value: form.ttl,
            onInput: (e: Event) => { form.ttl = eventTargetValue(e); },
          }),
        ),
        el('div', { className: 'cluster--actions' },
          renderButton({
            label: 'Dry-run preview',
            variant: 'secondary',
            size: 'sm',
            onClick: dryRun,
          }),
          renderButton({
            label: busy ? 'Blocking…' : 'Block IP',
            variant: 'danger',
            size: 'sm',
            loading: busy,
            disabled: busy || !form.ip.trim(),
            onClick: block,
          }),
        ),
        preview
          ? el('pre', { className: 'code-block text-sm mt-2' }, JSON.stringify(preview, null, 2))
          : null,
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
        el('table', { className: 'data-table', 'aria-label': 'Blacklist entries' },
          el('thead', null,
            el('tr', null,
              el('th', { scope: 'col' }, 'IP'),
              el('th', { scope: 'col' }, 'Reason'),
              el('th', { scope: 'col' }, 'Created'),
              el('th', { scope: 'col' }, 'Expires'),
              el('th', { scope: 'col' }, ''),
            ),
          ),
          el('tbody', null,
            loading ? tableSkeletonRows(5) : null,
            !loading && items.length === 0
              ? el('tr', null,
                renderEmptyTableCell(5, {
                  title: 'No blacklist entries',
                  description: 'Blocked IPs appear here after you add one above.',
                  icon: 'shield',
                }),
              )
              : null,
            items.map((row: BlacklistEntryDTO) => el('tr', null,
              el('td', { className: 'font-mono' }, row.ip ?? '—'),
              el('td', null, row.reason ?? '—'),
              el('td', null, row.created_at ? new Date(row.created_at).toLocaleString() : '—'),
              el('td', null, row.expires_at ? new Date(row.expires_at).toLocaleString() : '—'),
              el('td', null,
                renderButton({
                  label: 'Unblock',
                  variant: 'secondary',
                  size: 'sm',
                  onClick: () => unblock(row.ip, row.reason ?? 'manual'),
                }),
              ),
            )),
          ),
        ),
      ),
    );
  }

  load();
  return { destroy() { destroyed = true; } };
}
