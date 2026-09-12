import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { MonthPicker } from '@/components/ui/datetime_picker';
import type { BillingStatement } from '@/api/types';
import { CustomerDetailPanel } from '@/domains/customers/customer_detail_panel';
import { CustomerDetailRow } from '@/domains/customers/customer_detail_row';
import { SecondaryActionButton } from '@/shell/action_buttons';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import { CustomerTabShell } from '@/shell/customer_tab_shell';
import { StubBanner } from '@/shell/stub_banner';
import { INLINE_FILTER_ACTION_GRID_CLASS, FilterField } from '@/shell/filter_panel';
import { displayMicro } from '@/lib/display';
import { adminTypography } from '@/lib/admin_kit';

export type CustomerDetailStatementTabProps = {
  statementMonth: string;
  onStatementMonthChange: (month: string) => void;
  onStatementLoad: () => void;
  statement: BillingStatement | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
};

export function CustomerDetailStatementTab({
  statementMonth,
  onStatementMonthChange,
  onStatementLoad,
  statement,
  fetching,
  error,
  hasSnapshot,
}: CustomerDetailStatementTabProps) {
  const lines = statement?.lines ?? [];

  return (
    <section className="grid gap-4" >
      <div className={INLINE_FILTER_ACTION_GRID_CLASS} >
        <FilterField htmlFor="statement-month" label="Billing month">
          <MonthPicker
            id="statement-month"
            value={statementMonth}
            onChange={onStatementMonthChange}
          />
        </FilterField>
        <SecondaryActionButton
          disabled={fetching || !statementMonth}
          loading={fetching}
          type="button"
          onClick={onStatementLoad}
        >
          Load
        </SecondaryActionButton>
      </div>

      <CustomerTabShell
        blockingErrorTitle="Could not load statement"
        fetchState={{ fetching, error, hasSnapshot }}
      >
        {hasSnapshot && statement ? (
          <section>
            {'stale' in statement && statement.stale === true ? (
              <StubBanner
                message="Statement analytics may be delayed while ClickHouse catches up. Ledger totals and reconciliation remain Postgres-backed."
                title="Stale analytics"
              />
            ) : null}
            <Card>
              <CardHeader>
                <CardTitle>Statement summary</CardTitle>
              </CardHeader>
              <CardContent>
                <CustomerDetailPanel>
                  <CustomerDetailRow label="Currency" value={statement.currency} />
                  <CustomerDetailRow label="Period from" value={statement.period?.from} />
                  <CustomerDetailRow label="Period to" value={statement.period?.to} />
                  <CustomerDetailRow
                    label="Opening balance (micro)"
                    value={displayMicro(statement.opening_balance_micro)}
                  />
                  <CustomerDetailRow
                    label="Closing balance (micro)"
                    value={displayMicro(statement.closing_balance_micro)}
                  />
                  <CustomerDetailRow
                    label="Tax (micro)"
                    value={displayMicro(statement.tax_breakdown?.tax_micro)}
                  />
                  <CustomerDetailRow
                    label="Invoice total (micro)"
                    value={displayMicro(statement.reconciliation?.invoice_total_micro)}
                  />
                  <CustomerDetailRow
                    label="Ledger sum (micro)"
                    value={displayMicro(statement.reconciliation?.ledger_sum_micro)}
                  />
                  <CustomerDetailRow
                    label="Reconciliation delta (micro)"
                    value={displayMicro(statement.reconciliation?.delta_micro)}
                  />
                </CustomerDetailPanel>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Statement lines</CardTitle>
              </CardHeader>
              <CardContent className="overflow-x-auto" >
                {lines.length === 0 ? (
                  <p className={adminTypography.bodyMuted} >No statement lines for this month.</p>
                ) : (
                  <DirectoryTable nested>
                    <TableHeader>
                      <TableRow>
                        <DirectoryTableHead>Ledger type</DirectoryTableHead>
                        <DirectoryTableHead align="end">Amount (micro)</DirectoryTableHead>
                        <DirectoryTableHead align="end">Entry count</DirectoryTableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {lines.map((line, index) => (
                        <TableRow key={`${line.ledger_type ?? 'line'}-${index}`}>
                          <TableCell>{line.ledger_type ?? ''}</TableCell>
                          <TableCell className="text-right tabular-nums" >
                            {displayMicro(line.amount_micro)}
                          </TableCell>
                          <TableCell className="text-right tabular-nums" >{line.entry_count ?? ''}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </DirectoryTable>
                )}
              </CardContent>
            </Card>
          </section>
        ) : null}

        {!hasSnapshot && !fetching && !error ? (
          <p className={adminTypography.bodyMuted} >Choose a month and click Load.</p>
        ) : null}
      </CustomerTabShell>
    </section>
  );
}
