import type { BillingStatement } from '../../helpers/customers_api.js';
import type { SelfServeInvoice } from '../../helpers/selfserve_api.js';
import { formatAmountMicro } from '../../helpers/money.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import { PaginationBar } from '../system/pagination_bar.js';
import { EmptyState } from '../system/empty_state.js';
import styles from './selfserve_shared.module.css';

export type BillingPanelProps = {
  month: string;
  statement: BillingStatement | null;
  statementLoading: boolean;
  statementError: unknown;
  invoices: SelfServeInvoice[];
  invoiceTotal: number;
  limit: number;
  offset: number;
  invoicesLoading: boolean;
  invoicesError: unknown;
  onMonthChange: (month: string) => void;
  onOffsetChange: (offset: number) => void;
};

export function BillingPanel({
  month,
  statement,
  statementLoading,
  statementError,
  invoices,
  invoiceTotal,
  limit,
  offset,
  invoicesLoading,
  invoicesError,
  onMonthChange,
  onOffsetChange,
}: BillingPanelProps) {
  if (statementLoading && !statement && invoicesLoading && invoices.length === 0) {
    return <PageSkeleton rows={6} />;
  }

  return (
    <div data-testid="selfserve-billing-panel">
      <PageChrome title="Self-serve billing" badge={<span>{invoiceTotal} invoices</span>} />
      <p className={styles.intro}>
        Statement from GET /api/v1/selfserve/billing/statement. Invoice history from GET
        /api/v1/selfserve/invoices.
      </p>
      <div className={styles.form}>
        <label className={styles.field}>
          <span className={styles.label}>Statement month (YYYY-MM)</span>
          <input
            className={styles.input}
            value={month}
            onChange={(e) => onMonthChange(e.target.value)}
          />
        </label>
      </div>
      {statementError ? (
        <ErrorBlock error={statementError} fallbackTitle="Failed to load billing statement" />
      ) : statement ? (
        <dl className={styles.dl}>
          <dt>Opening</dt>
          <dd>{formatAmountMicro(statement.opening_balance_micro)}</dd>
          <dt>Closing</dt>
          <dd>{formatAmountMicro(statement.closing_balance_micro)}</dd>
        </dl>
      ) : statementLoading ? (
        <PageSkeleton rows={2} />
      ) : null}
      {statement?.lines && statement.lines.length > 0 ? (
        <div className={styles.grid} role="grid" aria-label="Statement lines">
          <div className={styles.gridHead} role="row">
            <span role="columnheader">Description</span>
            <span role="columnheader">Amount</span>
            <span role="columnheader" />
            <span role="columnheader" />
            <span role="columnheader" />
            <span role="columnheader" />
            <span role="columnheader" />
          </div>
          {statement.lines.map((line, index) => {
            const label =
              line.description ??
              (line as { ledger_type?: string }).ledger_type ??
              '-';
            return (
              <div key={`${label}-${index}`} className={styles.gridRow} role="row">
                <span role="gridcell">{label}</span>
                <span role="gridcell">{formatAmountMicro(line.amount_micro)}</span>
                <span role="gridcell" />
                <span role="gridcell" />
                <span role="gridcell" />
                <span role="gridcell" />
                <span role="gridcell" />
              </div>
            );
          })}
        </div>
      ) : null}
      <h3 className={styles.kpiLabel}>Invoices</h3>
      {invoicesError ? (
        <ErrorBlock error={invoicesError} fallbackTitle="Failed to load invoices" />
      ) : invoices.length === 0 && !invoicesLoading ? (
        <EmptyState message="No invoices returned for this page." />
      ) : (
        <div className={styles.grid} role="grid" aria-label="Invoices">
          <div className={styles.gridHead} role="row">
            <span role="columnheader">Month</span>
            <span role="columnheader">Status</span>
            <span role="columnheader">Total</span>
            <span role="columnheader" />
            <span role="columnheader" />
            <span role="columnheader" />
            <span role="columnheader" />
          </div>
          {invoices.map((invoice) => (
            <div key={invoice.id ?? invoice.billing_month} className={styles.gridRow} role="row">
              <span role="gridcell">{invoice.billing_month ?? '-'}</span>
              <span role="gridcell">{invoice.status ?? '-'}</span>
              <span role="gridcell">{formatAmountMicro(invoice.total_micro)}</span>
              <span role="gridcell" />
              <span role="gridcell" />
              <span role="gridcell" />
              <span role="gridcell" />
            </div>
          ))}
        </div>
      )}
      <div className={styles.footer}>
        <PaginationBar limit={limit} offset={offset} total={invoiceTotal} onOffsetChange={onOffsetChange} />
      </div>
    </div>
  );
}
