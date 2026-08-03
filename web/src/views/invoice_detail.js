import { el, replaceChildren } from '../lib/dom.js';
import { createResource } from '../lib/fetch_resource.js';
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
import { renderBreadcrumbs } from '../ui/breadcrumbs.js';
import { shortCustomerId } from '../helpers/customer_context.js';
import { renderStatusBadge } from '../ui/status_badge.js';
import { renderIcon } from '../ui/icon.js';
import { displayLabel } from '../helpers/display_labels.js';

/**
 * Mount the invoice detail view with void action.
 *
 * @param {HTMLElement} container
 * @param {{ params: Record<string, string> }} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container, ctx) {
  let destroyed = false;
  const id = ctx.params.id;
  let voidLoading = false;
  let lastError = null;

  const user = auth.getUser();
  const canVoid = can(user?.permissions ?? [], 'customers:write');

  const state = { data: null, loading: true, error: null };

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

    const invoiceCrumbs = [{ label: 'Billing', href: '/billing' }];
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
            el('button', {
              type: 'button',
              className: 'btn btn--secondary btn--sm',
              onClick: openPdf,
            },
              renderIcon('download', { size: 14 }),
              'PDF',
            ),
            canVoid
              ? el('button', {
                type: 'button',
                className: 'btn btn--danger btn--sm',
                disabled: voidLoading,
                onClick: handleVoid,
              },
                renderIcon('trash', { size: 14 }),
                'Void',
              )
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
      invoice.lines?.length > 0
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
              invoice.lines.map((line, _idx) =>
                el('tr', null,
                  el('td', null, displayLabel(line.ledger_type)),
                  el('td', { className: 'font-mono' }, String(line.amount_micro)),
                  el('td', null, String(line.entry_count)),
                ),
              ),
            ),
          ),
        )
        : null,
    );
  }

  const resource = createResource(
    () => `/api/v1/billing/invoices/${id}`,
    {
      onUpdate: (s) => {
        if (s.error !== lastError) {
          lastError = s.error;
          if (s.error) surfaceServiceErrorToast(s.error);
        }
        Object.assign(state, s);
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
