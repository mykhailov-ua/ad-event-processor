import type { RouteContext, ViewHandle } from '../lib/router_types.js';
import { el, replaceChildren, eventTargetValue } from '../lib/dom.js';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { isCustomerUuid } from '../helpers/customer_context.js';
import { hasBoundCustomer, boundCustomerId } from '../helpers/buyer_session.js';
import { scanMarginBreaches, type MarginBreachRow } from '../helpers/margin_guard_api.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderStatusBadge } from '../ui/status_badge.js';
import { tableSkeletonRows } from '../ui/data_table.js';
import { formatMicro } from '../helpers/money.js';
import { renderButton, renderButtonLink } from '../ui/button.js';

/**
 * Margin Guard portfolio — lists ACTIVE campaigns with margin_breach=true.
 * Canonical path: /margin-guard (alias: /integrations/margin-guard).
 */
export function mount(container: HTMLElement, ctx: RouteContext): ViewHandle {
  let destroyed = false;
  const user = auth.getUser();
  const canWrite = can(user?.permissions ?? [], 'campaigns:write');
  const sessionScoped = hasBoundCustomer(user?.role);
  const tenantCustomerId = boundCustomerId(user);

  let customerId = sessionScoped ? tenantCustomerId : (ctx.query.get('customer_id')?.trim() ?? '');
  let rows: MarginBreachRow[] = [];
  let loading = false;
  let error: Error | string | null = null;
  let scanned = false;

  async function scan() {
    if (!isCustomerUuid(customerId)) {
      rows = [];
      scanned = false;
      render();
      return;
    }
    loading = true;
    error = null;
    render();
    const result = await scanMarginBreaches(customerId);
    if (destroyed) return;
    loading = false;
    scanned = true;
    if (result.error) {
      error = result.error;
      rows = [];
    } else {
      rows = result.rows;
    }
    if (!sessionScoped && isCustomerUuid(customerId)) {
      try {
        const url = new URL(window.location.href);
        url.searchParams.set('customer_id', customerId);
        window.history.replaceState(null, '', url.pathname + url.search);
      } catch {
        // ignore
      }
    }
    render();
  }

  function render() {
    if (destroyed) return;
    if (error) {
      replaceChildren(container, renderErrorBlock(error, 'Margin scan failed'));
      return;
    }

    const customerField = !sessionScoped
      ? el('label', { className: 'form-field', htmlFor: 'mg-portfolio-customer' },
        'Customer UUID',
        el('input', {
          id: 'mg-portfolio-customer',
          className: 'form-input form-input--sm font-mono',
          value: customerId,
          'data-testid': 'margin-guard-customer',
          onInput: (e: Event) => { customerId = eventTargetValue(e).trim(); },
        }),
      )
      : el('p', { className: 'text-muted text-sm' },
        'Customer: ',
        el('span', { className: 'font-mono' }, customerId || '—'),
      );

    const emptyMsg = !isCustomerUuid(customerId)
      ? 'Enter customer UUID and scan.'
      : scanned
        ? 'No margin breaches in active campaigns.'
        : 'Click Scan to load breaches.';

    replaceChildren(container,
      el('section', { className: 'stack', 'data-testid': 'margin-guard-portfolio' },
        el('div', { className: 'page-header' },
          el('h1', { className: 'page-header__title' }, 'Margin Guard'),
          el('p', { className: 'page-header__desc' },
            'Active campaigns where RTB cost exceeds the revenue threshold (1h window). Open a row to edit policy.',
          ),
        ),
        customerField,
        el('div', { className: 'flex gap-2 items-center' },
          renderButton({
            label: loading ? 'Scanning…' : 'Scan campaigns',
            variant: 'secondary',
            size: 'sm',
            loading,
            disabled: loading || !isCustomerUuid(customerId),
            testId: 'margin-guard-scan',
            onClick: () => { void scan(); },
          }),
          rows.length > 0
            ? el('span', { className: 'text-muted text-sm', 'data-testid': 'margin-guard-count' },
              `${rows.length} breach${rows.length === 1 ? '' : 'es'}`,
            )
            : null,
        ),
        el('div', { className: 'table-wrapper elevation-raised' },
          el('table', { className: 'data-table' },
            el('thead', null,
              el('tr', null,
                el('th', null, 'Campaign'),
                el('th', null, 'Status'),
                el('th', null, 'Spend (1h)'),
                el('th', null, 'RTB cost'),
                el('th', null, 'Threshold (bps)'),
                el('th', null, ''),
              ),
            ),
            el('tbody', null,
              loading ? tableSkeletonRows(6) : null,
              !loading && rows.length === 0
                ? el('tr', null,
                  el('td', { colSpan: 6 }, emptyMsg),
                )
                : null,
              rows.map(({ campaign: c, margin: m }) => el('tr', null,
                el('td', null, c.name ?? c.id),
                el('td', null, renderStatusBadge(c.status)),
                el('td', { className: 'font-mono' }, '$' + formatMicro(m.advertiser_spend_micro ?? 0)),
                el('td', { className: 'font-mono' }, '$' + formatMicro(m.rtb_cost_micro ?? 0)),
                el('td', null, String(m.threshold_bps ?? '—')),
                el('td', null,
                  renderButtonLink({
                    label: canWrite ? 'Edit policy' : 'View',
                    href: `/campaigns/${c.id}?tab=margin`,
                    variant: 'secondary',
                    size: 'sm',
                    testId: 'margin-guard-edit-policy',
                    onClick: (e: Event) => {
                      e.preventDefault();
                      ctx.navigate(`/campaigns/${c.id}?tab=margin`);
                    },
                  }),
                ),
              )),
            ),
          ),
        ),
      ),
    );
  }

  if (isCustomerUuid(customerId)) void scan();
  else render();

  return {
    destroy() { destroyed = true; },
  };
}
