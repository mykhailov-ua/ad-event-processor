import { useState } from 'react';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ParseDecimal, formatAmountMicro } from '../helpers/money.js';
import {
  createPaymentIntent,
  fetchSelfServeStatement,
  formatPaymentStatus,
  type BillingStatementDTO,
  type PaymentIntentResult,
} from '../helpers/selfserve_billing_api.js';
import { to } from '../lib/to.js';
import { Button } from './button.js';
import { CopyableUuid } from './copyable_uuid.js';

export type BillingSelfServeSectionProps = {
  customerId: string;
  buyerMode: boolean;
};

export function BillingSelfServeSection({ customerId, buyerMode }: BillingSelfServeSectionProps) {
  const [amountInput, setAmountInput] = useState('100.00');
  const [topUpLoading, setTopUpLoading] = useState(false);
  const [paymentIntent, setPaymentIntent] = useState<PaymentIntentResult | null>(null);
  const [statementLoading, setStatementLoading] = useState(false);
  const [statement, setStatement] = useState<BillingStatementDTO | null>(null);
  const [statementMonth, setStatementMonth] = useState(() => new Date().toISOString().slice(0, 7));

  const submitTopUp = async () => {
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
    setTopUpLoading(true);
    setPaymentIntent(null);
    const [res, err] = await to(createPaymentIntent(amountMicro, customerId || undefined));
    setTopUpLoading(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      const view = mapServiceError(err);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      return;
    }
    setPaymentIntent(res ?? null);
    pushToastMessage({
      title: 'Payment intent created',
      message: formatPaymentStatus(res?.status ?? 'pending'),
    });
  };

  const loadStatement = async () => {
    setStatementLoading(true);
    const [data, err] = await to(fetchSelfServeStatement(statementMonth));
    setStatementLoading(false);
    if (err) {
      const view = mapServiceError(err);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      setStatement(null);
      return;
    }
    setStatement(data ?? null);
  };

  const cryptoDeposit = Boolean(paymentIntent?.deposit_address);

  return (
    <div className="stack" data-testid="billing-selfserve-panel">
      <section className="section-card stack">
        <h3 className="subsection-title">Wallet top-up</h3>
        <p className="text-muted text-sm">
          Creates a payment intent. Crypto checkouts include a USDT deposit address and QR code.
        </p>
        <label className="form-field" htmlFor="topup-amount">
          Amount (USD)
          <input
            id="topup-amount"
            className="form-input form-input--sm"
            inputMode="decimal"
            value={amountInput}
            data-testid="billing-topup-amount"
            onChange={(e) => setAmountInput(e.target.value)}
          />
        </label>
        <Button
          label={topUpLoading ? 'Creating...' : 'Create payment'}
          variant="primary"
          size="sm"
          loading={topUpLoading}
          disabled={topUpLoading}
          data-testid="billing-topup-submit"
          onClick={() => void submitTopUp()}
        />
        {paymentIntent ? (
          <div className="stack mt-2" data-testid="billing-topup-result">
            <dl className="definition-list">
              <dt>Status</dt>
              <dd data-testid="billing-topup-status">
                {formatPaymentStatus(paymentIntent.status)}
              </dd>
              {paymentIntent.provider_ref ? (
                <>
                  <dt>Reference</dt>
                  <dd className="font-mono text-sm">{paymentIntent.provider_ref}</dd>
                </>
              ) : null}
            </dl>
            {cryptoDeposit ? (
              <div className="stack" data-testid="billing-crypto-deposit">
                <p className="text-sm text-muted">
                  {paymentIntent.deposit_network ?? 'USDT TRC-20'}
                </p>
                <dl className="definition-list">
                  <dt>Deposit address</dt>
                  <dd>
                    <span className="font-mono text-sm" data-testid="billing-deposit-address">
                      {paymentIntent.deposit_address}
                    </span>
                    <div className="mt-1">
                      <CopyableUuid uuid={paymentIntent.deposit_address ?? ''} />
                    </div>
                  </dd>
                </dl>
                {paymentIntent.deposit_qr_svg ? (
                  <div
                    className="deposit-qr"
                    data-testid="billing-deposit-qr"
                    dangerouslySetInnerHTML={{ __html: paymentIntent.deposit_qr_svg }}
                  />
                ) : null}
              </div>
            ) : null}
            {paymentIntent.checkout_url ? (
              <p className="text-sm">
                <a
                  href={paymentIntent.checkout_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  data-testid="billing-topup-checkout-link"
                >
                  Open checkout {'->'}
                </a>
              </p>
            ) : null}
          </div>
        ) : null}
      </section>
      {buyerMode ? (
        <section className="section-card stack">
          <h3 className="subsection-title">Billing statement</h3>
          <label className="form-field" htmlFor="stmt-month">
            Month (YYYY-MM)
            <input
              id="stmt-month"
              className="form-input form-input--sm"
              value={statementMonth}
              onChange={(e) => setStatementMonth(e.target.value)}
            />
          </label>
          <Button
            label={statementLoading ? 'Loading...' : 'Load statement'}
            variant="secondary"
            size="sm"
            loading={statementLoading}
            disabled={statementLoading}
            data-testid="billing-statement-load"
            onClick={() => void loadStatement()}
          />
          {statement ? (
            <dl className="definition-list mt-2">
              <dt>Opening</dt>
              <dd className="font-mono">
                {formatAmountMicro(statement.opening_balance_micro ?? 0, statement.currency)}
              </dd>
              <dt>Closing</dt>
              <dd className="font-mono">
                {formatAmountMicro(statement.closing_balance_micro ?? 0, statement.currency)}
              </dd>
              <dt>Period</dt>
              <dd>{`${statement.period?.from ?? '-'} -> ${statement.period?.to ?? '-'}`}</dd>
            </dl>
          ) : null}
        </section>
      ) : null}
    </div>
  );
}
