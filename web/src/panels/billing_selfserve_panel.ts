import type { ViewHandle } from '../lib/router_types.js';
import { el, eventTargetValue, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ParseDecimal, formatAmountMicro } from '../helpers/money.js';
import {
  createPaymentIntent,
  fetchSelfServeStatement,
  type BillingStatementDTO,
} from '../helpers/selfserve_billing_api.js';
import { renderButton } from '../ui/button.js';

export type BillingSelfServePanelOpts = {
  customerId: string;
  buyerMode: boolean;
};

/**
 * Mount wallet top-up and self-serve statement panel for buyers/tenants.
 */
export function mountBillingSelfServePanel(
  container: HTMLElement,
  opts: BillingSelfServePanelOpts,
): ViewHandle {
  let destroyed = false;
  let amountInput = '100.00';
  let topUpLoading = false;
  let checkoutUrl = '';
  let statementLoading = false;
  let statement: BillingStatementDTO | null = null;
  let statementMonth = new Date().toISOString().slice(0, 7);

  function render(): void {
    if (destroyed) return;
    replaceChildren(container,
      el('div', { className: 'stack', 'data-testid': 'billing-selfserve-panel' },
        el('section', { className: 'section-card stack' },
          el('h3', { className: 'subsection-title' }, 'Wallet top-up'),
          el('p', { className: 'text-muted text-sm' },
            'Creates a payment intent and checkout URL (Stripe or configured provider).',
          ),
          el('label', { className: 'form-field', htmlFor: 'topup-amount' },
            'Amount (USD)',
            el('input', {
              id: 'topup-amount',
              className: 'form-input form-input--sm',
              inputMode: 'decimal',
              value: amountInput,
              'data-testid': 'billing-topup-amount',
              onInput: (e: Event) => { amountInput = eventTargetValue(e); },
            }),
          ),
          renderButton({
            label: topUpLoading ? 'Creating…' : 'Create payment',
            variant: 'primary',
            size: 'sm',
            loading: topUpLoading,
            disabled: topUpLoading,
            testId: 'billing-topup-submit',
            onClick: submitTopUp,
          }),
          checkoutUrl
            ? el('p', { className: 'text-sm mt-2' },
              el('a', {
                href: checkoutUrl,
                target: '_blank',
                rel: 'noopener noreferrer',
                'data-testid': 'billing-topup-checkout-link',
              }, 'Open checkout →'),
            )
            : null,
        ),
        opts.buyerMode
          ? el('section', { className: 'section-card stack' },
            el('h3', { className: 'subsection-title' }, 'Billing statement'),
            el('label', { className: 'form-field', htmlFor: 'stmt-month' },
              'Month (YYYY-MM)',
              el('input', {
                id: 'stmt-month',
                className: 'form-input form-input--sm',
                value: statementMonth,
                onChange: (e: Event) => { statementMonth = eventTargetValue(e); },
              }),
            ),
            renderButton({
              label: statementLoading ? 'Loading…' : 'Load statement',
              variant: 'secondary',
              size: 'sm',
              loading: statementLoading,
              disabled: statementLoading,
              testId: 'billing-statement-load',
              onClick: loadStatement,
            }),
            statement
              ? el('dl', { className: 'definition-list mt-2' },
                el('dt', null, 'Opening'),
                el('dd', { className: 'font-mono' },
                  formatAmountMicro(statement.opening_balance_micro ?? 0, statement.currency),
                ),
                el('dt', null, 'Closing'),
                el('dd', { className: 'font-mono' },
                  formatAmountMicro(statement.closing_balance_micro ?? 0, statement.currency),
                ),
                el('dt', null, 'Period'),
                el('dd', null,
                  `${statement.period?.from ?? '—'} → ${statement.period?.to ?? '—'}`,
                ),
              )
              : null,
          )
          : null,
      ),
    );
  }

  async function submitTopUp(): Promise<void> {
    if (topUpLoading) return;
    let amountMicro: number;
    try {
      amountMicro = ParseDecimal(amountInput.trim());
    } catch {
      pushToastMessage({ title: 'Invalid amount', message: 'Enter a positive decimal amount' });
      return;
    }
    if (amountMicro <= 0) {
      pushToastMessage({ title: 'Invalid amount', message: 'Amount must be greater than zero' });
      return;
    }
    topUpLoading = true;
    checkoutUrl = '';
    render();
    const [res, err] = await to(createPaymentIntent(
      amountMicro,
      opts.customerId || undefined,
    ));
    topUpLoading = false;
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
    checkoutUrl = res?.checkout_url ?? '';
    pushToastMessage({ title: 'Payment intent created', message: res?.status ?? 'pending' });
    render();
  }

  async function loadStatement(): Promise<void> {
    statementLoading = true;
    render();
    const [data, err] = await to(fetchSelfServeStatement(statementMonth));
    statementLoading = false;
    if (destroyed) return;
    if (err) {
      const view = mapServiceError(err);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      statement = null;
    } else {
      statement = data ?? null;
    }
    render();
  }

  render();

  return {
    destroy() {
      destroyed = true;
    },
  };
}
