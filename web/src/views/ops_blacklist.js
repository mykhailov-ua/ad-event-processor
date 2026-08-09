import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import { apiConfirmed } from '../helpers/confirmed_api.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { tableSkeletonRows } from '../ui/data_table.js';

const PAGE_SIZE = 50;

/**
 * Mount ops IP blacklist management UI.
 *
 * @param {HTMLElement} container
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container) {
  let destroyed = false;
  let page = 0;
  let items = [];
  let total = 0;
  let loading = true;
  let error = null;
  let preview = null;
  const form = { ip: '', source: 'manual', ttl: '' };
  let busy = false;

  async function load() {
    loading = true;
    render();
    const offset = page * PAGE_SIZE;
    const [res, err] = await to(api(`/api/v1/ops/blacklist?limit=${PAGE_SIZE}&offset=${offset}`));
    if (destroyed) return;
    loading = false;
    if (err) {
      error = err;
      render();
      return;
    }
    error = null;
    items = res?.data?.items ?? [];
    total = res?.data?.total ?? items.length;
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

  async function unblock(ip, source) {
    const [, err] = await to(apiConfirmed('/api/v1/ops/blacklist', {
      method: 'DELETE',
      body: JSON.stringify({ ip, source }),
    }));
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Unblock failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({ title: 'IP unblocked', message: ip });
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
            onInput: (e) => { form.ip = e.target.value; },
          }),
        ),
        el('label', { className: 'form-field', htmlFor: 'bl-source' },
          'Source',
          el('input', {
            id: 'bl-source',
            className: 'form-input form-input--sm',
            value: form.source,
            onInput: (e) => { form.source = e.target.value; },
          }),
        ),
        el('label', { className: 'form-field', htmlFor: 'bl-ttl' },
          'TTL seconds (optional)',
          el('input', {
            id: 'bl-ttl',
            className: 'form-input form-input--sm',
            inputMode: 'numeric',
            value: form.ttl,
            onInput: (e) => { form.ttl = e.target.value; },
          }),
        ),
        el('div', { className: 'flex gap-2' },
          el('button', {
            type: 'button',
            className: 'btn btn--secondary btn--sm',
            onClick: dryRun,
          }, 'Dry-run preview'),
          el('button', {
            type: 'button',
            className: 'btn btn--danger btn--sm',
            disabled: busy || !form.ip.trim(),
            onClick: block,
          }, busy ? 'Blocking…' : 'Block IP'),
        ),
        preview
          ? el('pre', { className: 'code-block text-sm mt-2' }, JSON.stringify(preview, null, 2))
          : null,
      ),
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
              ? el('tr', null, el('td', { colSpan: 5 }, 'No blacklist entries.'))
              : null,
            items.map((row) => el('tr', null,
              el('td', { className: 'font-mono' }, row.ip ?? '—'),
              el('td', null, row.reason ?? '—'),
              el('td', null, row.created_at ? new Date(row.created_at).toLocaleString() : '—'),
              el('td', null, row.expires_at ? new Date(row.expires_at).toLocaleString() : '—'),
              el('td', null,
                el('button', {
                  type: 'button',
                  className: 'btn btn--secondary btn--sm',
                  onClick: () => unblock(row.ip, row.reason ?? 'manual'),
                }, 'Unblock'),
              ),
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
  return { destroy() { destroyed = true; } };
}
