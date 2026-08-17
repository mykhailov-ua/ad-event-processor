import { useEffect, useState } from 'react';
import * as auth from '../helpers/auth.js';
import { boundCustomerId } from '../helpers/buyer_session.js';
import { BillingSelfServeSection } from '../components/billing_selfserve_section.js';
import { fetchSelfServeInvoices } from '../helpers/selfserve_api.js';
import { to } from '../lib/to.js';
import { ErrorBlock } from '../components/error_block.js';

/**
 * Self-serve billing: wallet top-up, statement, invoices.
 */
export function SelfServeBillingPage() {
  const customerId = boundCustomerId(auth.getUser());
  const [invoices, setInvoices] = useState<Array<{ id: string; status?: string; created_at?: string }>>([]);
  const [error, setError] = useState<unknown>(null);

  useEffect(() => {
    void (async () => {
      const [data, err] = await to(fetchSelfServeInvoices());
      if (err) {
        setError(err);
        return;
      }
      setInvoices(data ?? []);
    })();
  }, []);

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Billing unavailable" />;
  }

  return (
    <section className="stack" data-testid="selfserve-billing-page">
      <div className="page-header">
        <h1 className="page-header__title">Billing</h1>
        <p className="page-header__desc">Wallet top-up, monthly statement, and invoices.</p>
      </div>
      <BillingSelfServeSection customerId={customerId} buyerMode />
      <div className="section-card">
        <h2 className="subsection-title">Invoices</h2>
        <div className="table-wrapper">
          <table className="data-table" data-testid="selfserve-invoices-table">
            <thead>
              <tr>
                <th>ID</th>
                <th>Status</th>
                <th>Created</th>
              </tr>
            </thead>
            <tbody>
              {invoices.map((inv) => (
                <tr key={inv.id}>
                  <td className="font-mono text-sm">{inv.id}</td>
                  <td>{inv.status ?? '—'}</td>
                  <td>{inv.created_at ?? '—'}</td>
                </tr>
              ))}
              {invoices.length === 0 ? (
                <tr><td colSpan={3} className="text-muted">No invoices yet.</td></tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  );
}
