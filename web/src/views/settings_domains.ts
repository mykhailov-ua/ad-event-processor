import type { ViewHandle } from '../lib/router_types.js';
import { el, replaceChildren, eventTargetValue } from '../lib/dom.js';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderStatusBadge } from '../ui/status_badge.js';
import { tableSkeletonRows } from '../ui/data_table.js';
import { renderButton } from '../ui/button.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import {
  type DomainHealthRow,
  addCustomDomain,
  deleteCustomDomain,
  fetchDomains,
  healthStatusLabel,
  probeDomain,
  setupDomainSSL,
  sslStatusLabel,
} from '../helpers/domains_api.js';

function healthBadgeClass(status: string): string {
  switch (status) {
    case 'healthy': return 'ACTIVE';
    case 'degraded': return 'PAUSED';
    case 'down': return 'FAILED';
    default: return 'UNKNOWN';
  }
}

/**
 * Domain health and SSL setup for tracking/admin/custom hosts.
 */
export function mount(container: HTMLElement): ViewHandle {
  let destroyed = false;
  const user = auth.getUser();
  const canWrite = can(user?.permissions ?? [], 'settings:write');
  let rows: DomainHealthRow[] = [];
  let loading = true;
  let error: Error | string | null = null;
  let busy = false;
  let customHost = '';

  async function reload() {
    loading = true;
    error = null;
    render();
    try {
      rows = await fetchDomains();
    } catch (e) {
      error = e instanceof Error ? e : String(e);
    } finally {
      if (!destroyed) {
        loading = false;
        render();
      }
    }
  }

  async function addCustom() {
    if (!canWrite || !customHost.trim()) return;
    busy = true;
    render();
    try {
      await addCustomDomain(customHost.trim());
      customHost = '';
      pushToastMessage({ title: 'Domain added', message: 'Custom domain registered for probing' });
      await reload();
    } catch (e) {
      if (e instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Add failed', message: mapServiceError(e).message });
    } finally {
      busy = false;
      if (!destroyed) render();
    }
  }

  async function removeHost(hostname: string) {
    if (!canWrite) return;
    busy = true;
    render();
    try {
      await deleteCustomDomain(hostname);
      pushToastMessage({ title: 'Domain removed', message: hostname });
      await reload();
    } catch (e) {
      if (e instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Delete failed', message: mapServiceError(e).message });
    } finally {
      busy = false;
      if (!destroyed) render();
    }
  }

  async function probeNow(hostname: string) {
    if (!canWrite) return;
    busy = true;
    render();
    try {
      await probeDomain(hostname);
      pushToastMessage({ title: 'Probe complete', message: hostname });
      await reload();
    } catch (e) {
      if (e instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Probe failed', message: mapServiceError(e).message });
    } finally {
      busy = false;
      if (!destroyed) render();
    }
  }

  async function setupSSL(hostname: string) {
    if (!canWrite) return;
    busy = true;
    render();
    try {
      const result = await setupDomainSSL(hostname);
      pushToastMessage({
        title: result.status === 'ok' ? 'SSL setup started' : 'SSL setup failed',
        message: result.message,
      });
      await reload();
    } catch (e) {
      if (e instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'SSL setup failed', message: mapServiceError(e).message });
    } finally {
      busy = false;
      if (!destroyed) render();
    }
  }

  function renderTable() {
    if (loading) {
      return el('section', { className: 'card', 'data-testid': 'domains-table' }, tableSkeletonRows(3));
    }
    return el('section', { className: 'card stack', 'data-testid': 'domains-table' },
      el('h2', { className: 'h3' }, 'Monitored domains'),
      rows.length === 0
        ? el('p', { className: 'text-muted' }, 'No domains configured yet. Set tracking domain in Platform settings.')
        : el('table', { className: 'data-table' },
          el('thead', {},
            el('tr', {},
              el('th', {}, 'Hostname'),
              el('th', {}, 'Role'),
              el('th', {}, 'Health'),
              el('th', {}, 'SSL'),
              el('th', {}, 'Latency'),
              el('th', {}, 'Last probe'),
              el('th', {}, ''),
            ),
          ),
          el('tbody', {},
            ...rows.map((row) =>
              el('tr', { 'data-testid': `domain-row-${row.hostname}` },
                el('td', {}, row.hostname),
                el('td', {}, row.role),
                el('td', {}, renderStatusBadge(healthBadgeClass(row.health_status), { label: healthStatusLabel(row.health_status) })),
                el('td', {},
                  renderStatusBadge(row.ssl_status.toUpperCase(), { label: sslStatusLabel(row.ssl_status), kind: 'service' }),
                  row.ssl_not_after
                    ? el('div', { className: 'text-muted text-sm' }, `until ${new Date(row.ssl_not_after).toLocaleDateString()}`)
                    : null,
                ),
                el('td', {}, row.probe_latency_ms != null ? `${row.probe_latency_ms} ms` : '—'),
                el('td', {}, row.last_probe_at ? new Date(row.last_probe_at).toLocaleString() : '—'),
                el('td', { className: 'row gap-xs' },
                  canWrite
                    ? renderButton({
                      label: 'Probe',
                      variant: 'ghost',
                      size: 'sm',
                      disabled: busy,
                      testId: 'domain-probe',
                      onClick: () => { void probeNow(row.hostname); },
                    })
                    : null,
                  canWrite
                    ? renderButton({
                      label: 'Setup SSL',
                      variant: 'ghost',
                      size: 'sm',
                      disabled: busy,
                      testId: 'domain-ssl-setup',
                      onClick: () => { void setupSSL(row.hostname); },
                    })
                    : null,
                  canWrite && row.role === 'custom'
                    ? renderButton({
                      label: 'Remove',
                      variant: 'ghost',
                      size: 'sm',
                      disabled: busy,
                      onClick: () => { void removeHost(row.hostname); },
                    })
                    : null,
                ),
              ),
            ),
          ),
        ),
    );
  }

  function render() {
    if (destroyed) return;
    replaceChildren(container,
      el('header', { className: 'page-header' },
        el('h1', { className: 'h2' }, 'Domains'),
        el('p', { className: 'text-muted' },
          'Health probes every 5 minutes (HTTP + TLS). Tracking and admin hosts sync from platform config.',
        ),
      ),
      error ? renderErrorBlock(error) : null,
      canWrite
        ? el('section', { className: 'card stack', 'data-testid': 'domains-add-custom' },
          el('h2', { className: 'h3' }, 'Add custom domain'),
          el('div', { className: 'row gap-sm' },
            el('input', {
              type: 'text',
              className: 'form-input',
              placeholder: 'lander.example.com',
              value: customHost,
              disabled: busy,
              oninput: (e: Event) => { customHost = eventTargetValue(e); },
            }),
            renderButton({
              label: 'Add',
              variant: 'primary',
              disabled: busy || !customHost.trim(),
              onClick: () => { void addCustom(); },
            }),
          ),
        )
        : null,
      renderTable(),
    );
  }

  void reload();

  return {
    destroy() {
      destroyed = true;
    },
  };
}
