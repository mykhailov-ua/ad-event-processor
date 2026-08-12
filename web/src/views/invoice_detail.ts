import type { RouteContext, ViewHandle } from '../lib/router_types.js';
import type { InvoiceDTO, InvoiceLineDTO, InvoiceDeliveryDTO } from '../types/api/index.js';
import { el, replaceChildren } from '../lib/dom.js';
import { createResource, type ResourceState } from '../lib/fetch_resource.js';
import { to } from '../lib/to.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { apiConfirmed } from '../helpers/confirmed_api.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { can } from '../helpers/permissions.js';
import * as auth from '../helpers/auth.js';
import { formatAmountMicro } from '../helpers/money.js';
import {
  isPageBlockingError,
  mapServiceError,
} from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { surfaceServiceErrorToast } from '../helpers/service_error_toast.js';
import { renderBreadcrumbs, type BreadcrumbItem } from '../ui/breadcrumbs.js';
import { shortCustomerId } from '../helpers/customer_context.js';
import { renderStatusBadge } from '../ui/status_badge.js';
import { renderIcon } from '../ui/icon.js';
import { displayLabel } from '../helpers/display_labels.js';
import {
  fetchInvoiceDeliveries,
  retryInvoiceDelivery,
} from '../helpers/billing_admin_api.js';
import { renderButton } from '../ui/button.js';
import { renderEmptyTableCell } from '../ui/data_table.js';

/**
 * Mount the invoice detail view with void action.
 *
 * @param {HTMLElement} container
 * @param {{ params: Record<string, string> }} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container: HTMLElement, ctx: RouteContext): ViewHandle {
  let destroyed = false;
  const id = ctx.params.id;
  let voidLoading = false;
  let lastError: any = null;

  const user = auth.getUser();
  const canVoid = can(user?.permissions ?? [], 'customers:write');
  const canRetryDelivery = can(user?.permissions ?? [], 'customers:write');

  const state: ResourceState<InvoiceDTO> = { data: null, loading: true, error: null };
  let deliveries: InvoiceDeliveryDTO[] = [];
  let deliveriesLoading = false;
  let deliveryRetryLoading = false;

  async function handleVoid() {
    voidLoading = true;
    render();
    const [, voidErr] = await to(apiConfirmed(`/api/v1/billing/invoices/${id}/void`, { method: 'POST' }));
    if (voidErr) {
      if (voidErr instanceof ConfirmCancelledError) {
        voidLoading = false;
        render();
        return;
      }
      const view = mapServiceError(voidErr);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      voidLoading = false;
      render();
      return;
    }
    pushToastMessage({ title: 'Invoice voided', message: id });
    resource.reload();
    voidLoading = false;
    render();
  }

  function openPdf() {
    window.open(`/api/v1/billing/invoices/${id}/pdf`, '_blank', 'noopener,noreferrer');
  }

  async function loadDeliveries(): Promise<void> {
    deliveriesLoading = true;
    render();
    const [res, err] = await to(fetchInvoiceDeliveries(id));
    deliveriesLoading = false;
    if (destroyed) return;
    if (err) {
      deliveries = [];
    } else {
      deliveries = res?.items ?? [];
    }
    render();
  }

  async function handleDeliveryRetry(): Promise<void> {
    if (!canRetryDelivery || deliveryRetryLoading) return;
    deliveryRetryLoading = true;
    render();
    const [, err] = await to(retryInvoiceDelivery(id));
    deliveryRetryLoading = false;
    if (err) {
      if (err instanceof ConfirmCancelledError) {
        render();
        return;
      }
      const view = mapServiceError(err);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      render();
      return;
    }
    pushToastMessage({ title: 'Delivery retry queued', message: id });
    loadDeliveries();
  }

  function render() {
    if (destroyed) return;

    if (state.loading) {
      replaceChildren(container, el('span', { className: 'text-muted' }, 'Loading…'));
      return;
    }

    if (state.error) {
      const view = mapServiceError(state.error);
      if (isPageBlockingError(view) || view.kind === 'empty') {
        replaceChildren(container, renderErrorBlock(state.error));
        return;
      }
      replaceChildren(container);
      return;
    }

    const invoice = state.data;
    if (!invoice) {
      replaceChildren(container);
      return;
    }

    const invoiceCrumbs: BreadcrumbItem[] = [{ label: 'Billing', href: '/billing' }];
    if (invoice.customer_id) {
      invoiceCrumbs.push({
        label: shortCustomerId(invoice.customer_id, 12),
        href: `/customers/${invoice.customer_id}`,
      });
    }
    invoiceCrumbs.push({ label: shortCustomerId(id, 12) });

    replaceChildren(container,
      el('div', { className: 'page-header' },
        renderBreadcrumbs(invoiceCrumbs),
        el('div', { className: 'page-header__row' },
          el('div', { className: 'flex items-center gap-2' },
            renderIcon('file-text', { size: 20, className: 'text-muted' }),
            el('h1', { className: 'page-header__title' }, 'Invoice'),
          ),
          invoice.status
            ? renderStatusBadge(invoice.status, { kind: 'invoice' })
            : null,
          el('div', { className: 'flex items-center gap-2 ml-auto' },
            renderButton({
              label: 'PDF',
              variant: 'secondary',
              size: 'sm',
              icon: 'download',
              onClick: openPdf,
            }),
            canVoid && invoice.status !== 'VOID'
              ? renderButton({
                label: 'Void',
                variant: 'danger',
                size: 'sm',
                icon: 'trash',
                loading: voidLoading,
                disabled: voidLoading,
                onClick: handleVoid,
              })
              : null,
          ),
        ),
      ),
      el('div', { className: 'grid-stats section-block' },
        el('div', { className: 'metric-card' },
          el('div', { className: 'metric-card__label' }, 'Month'),
          el('div', { className: 'metric-card__value' }, invoice.billing_month ?? '—'),
        ),
        el('div', { className: 'metric-card' },
          el('div', { className: 'metric-card__label' }, 'Total'),
          el('div', { className: 'metric-card__value font-mono' },
            formatAmountMicro(invoice.total_micro ?? 0, invoice.currency),
          ),
        ),
        el('div', { className: 'metric-card' },
          el('div', { className: 'metric-card__label' }, 'Tax'),
          el('div', { className: 'metric-card__value font-mono' },
            formatAmountMicro(invoice.tax_micro ?? 0, invoice.currency),
          ),
        ),
        el('div', { className: 'metric-card' },
          el('div', { className: 'metric-card__label' }, 'Customer'),
          el('div', { className: 'metric-card__value font-mono' }, invoice.customer_id),
        ),
      ),
      (invoice.lines?.length ?? 0) > 0
        ? el('div', { className: 'table-wrapper elevation-raised mt-4' },
          el('table', { className: 'data-table' },
            el('thead', null,
              el('tr', null,
                el('th', { scope: 'col' }, 'Ledger type'),
                el('th', { scope: 'col' }, 'Amount (micro)'),
                el('th', { scope: 'col' }, 'Entries'),
              ),
            ),
            el('tbody', null,
              (invoice.lines ?? []).map((line: InvoiceLineDTO) =>
                el('tr', null,
                  el('td', null, displayLabel(line.ledger_type)),
                  el('td', { className: 'font-mono' }, String(line.amount_micro ?? 0)),
                  el('td', null, String(line.entry_count ?? 0)),
                ),
              ),
            ),
          ),
        )
        : null,
      el('section', { className: 'section-block', 'data-testid': 'invoice-deliveries' },
        el('div', { className: 'flex items-center gap-2 mb-3' },
          el('h2', { className: 'subsection-title' }, 'Delivery attempts'),
          canRetryDelivery && invoice.status !== 'VOID'
            ? renderButton({
              label: 'Retry delivery',
              variant: 'secondary',
              size: 'sm',
              className: 'ml-auto',
              loading: deliveryRetryLoading,
              disabled: deliveryRetryLoading,
              testId: 'invoice-delivery-retry',
              onClick: handleDeliveryRetry,
            })
            : null,
        ),
        deliveriesLoading ? el('span', { className: 'text-muted' }, 'Loading…') : null,
        el('div', { className: 'table-wrapper elevation-raised' },
          el('table', { className: 'data-table' },
            el('thead', null,
              el('tr', null,
                el('th', { scope: 'col' }, 'Channel'),
                el('th', { scope: 'col' }, 'Status'),
                el('th', { scope: 'col' }, 'Recipient'),
                el('th', { scope: 'col' }, 'Last error'),
                el('th', { scope: 'col' }, 'Sent'),
              ),
            ),
            el('tbody', null,
              deliveries.length === 0 && !deliveriesLoading
                ? el('tr', null,
                  renderEmptyTableCell(5, {
                    title: 'No deliveries yet',
                    description: 'Invoice delivery attempts will appear here after send.',
                  }),
                )
                : null,
              deliveries.map((d: InvoiceDeliveryDTO) => el('tr', null,
                el('td', null, displayLabel(d.provider)),
                el('td', null, d.status),
                el('td', { className: 'font-mono text-xs' }, d.recipient ?? '—'),
                el('td', { className: 'text-xs text-muted' }, d.error_message ?? '—'),
                el('td', { className: 'text-muted text-xs' },
                  d.updated_at ? new Date(d.updated_at).toLocaleString() : '—',
                ),
              )),
            ),
          ),
        ),
      ),
    );
  }

  const resource = createResource<InvoiceDTO>(
    () => `/api/v1/billing/invoices/${id}`,
    {
      onUpdate: (s) => {
        if (s.error !== lastError) {
          lastError = s.error;
          if (s.error) surfaceServiceErrorToast(s.error);
        }
        Object.assign(state, s);
        if (s.data && !deliveriesLoading && deliveries.length === 0) {
          loadDeliveries();
        }
        render();
      },
    },
  );

  render();

  return {
    destroy() {
      destroyed = true;
      resource.destroy();
    },
  };
}
