import { useEffect, useState } from 'react';

import { PageChrome } from '@/shell/page_chrome';
import { CustomerScopeBar } from '@/shell/customer_scope_bar';
import { EmptyState } from '@/shell/empty_state';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { APIKeyCreatedResponse, BillingStatement, Invoice, PaymentIntentCreatedResponse } from '@/api/types';
import { PortalsNav, portalsPanelError } from '@/domains/portals/portals_nav';
import { displayMicro } from '@/lib/display';

export type SelfServePortalProps = {
  invoices: Invoice[];
  appliedCustomerId: string;
  draftCustomerId: string;
  fetchingInvoices: boolean;
  invoicesError: Error | undefined;
  hasInvoicesSnapshot: boolean;
  draftPaymentAmountMicro: string;
  draftApiKeyName: string;
  draftPauseCampaignId: string;
  draftPauseReason: string;
  acting: boolean;
  actionError: Error | undefined;
  paymentResult: PaymentIntentCreatedResponse | undefined;
  apiKeyResult: APIKeyCreatedResponse | undefined;
  actionMessage: string | undefined;
  onDraftCustomerIdChange: (value: string) => void;
  onApplyCustomerScope: () => void;
  onDraftPaymentAmountMicroChange: (value: string) => void;
  onDraftApiKeyNameChange: (value: string) => void;
  onDraftPauseCampaignIdChange: (value: string) => void;
  onDraftPauseReasonChange: (value: string) => void;
  onCreatePaymentIntent: () => void;
  onCreateApiKey: () => void;
  onPauseCampaign: () => void;
  onResumeCampaign: () => void;
  draftStatementMonth: string;
  statement: BillingStatement | undefined;
  fetchingStatement: boolean;
  statementError: Error | undefined;
  onDraftStatementMonthChange: (value: string) => void;
  onLoadStatement: () => void;
};

export function SelfServePortal({
  invoices,
  appliedCustomerId,
  draftCustomerId,
  fetchingInvoices,
  invoicesError,
  hasInvoicesSnapshot,
  draftPaymentAmountMicro,
  draftApiKeyName,
  draftPauseCampaignId,
  draftPauseReason,
  acting,
  actionError,
  paymentResult,
  apiKeyResult,
  actionMessage,
  onDraftCustomerIdChange,
  onApplyCustomerScope,
  onDraftPaymentAmountMicroChange,
  onDraftApiKeyNameChange,
  onDraftPauseCampaignIdChange,
  onDraftPauseReasonChange,
  onCreatePaymentIntent,
  onCreateApiKey,
  onPauseCampaign,
  onResumeCampaign,
  draftStatementMonth,
  statement,
  fetchingStatement,
  statementError,
  onDraftStatementMonthChange,
  onLoadStatement,
}: SelfServePortalProps) {
  const [statementOpen, setStatementOpen] = useState(false);
  const [paymentOpen, setPaymentOpen] = useState(false);
  const [apiKeyOpen, setApiKeyOpen] = useState(false);
  const [pauseOpen, setPauseOpen] = useState(false);

  useEffect(() => {
    if (statement) {
      setStatementOpen(false);
    }
  }, [statement]);

  useEffect(() => {
    if (paymentResult) {
      setPaymentOpen(false);
    }
  }, [paymentResult]);

  useEffect(() => {
    if (apiKeyResult) {
      setApiKeyOpen(false);
    }
  }, [apiKeyResult]);

  if (!appliedCustomerId) {
    return (
      <PageChrome title="Self-serve">
        <PortalsNav />
        <CustomerScopeBar
          appliedCustomerId={appliedCustomerId}
          draftCustomerId={draftCustomerId}
          onApply={onApplyCustomerScope}
          onDraftCustomerIdChange={onDraftCustomerIdChange}
        />
        <EmptyState
          title="Customer required"
          description="Apply a customer ID for self-serve billing and campaign controls."
        />
      </PageChrome>
    );
  }

  return (
    <PageChrome title="Self-serve">
      <PortalsNav />

      <CustomerScopeBar
        appliedCustomerId={appliedCustomerId}
        draftCustomerId={draftCustomerId}
        onApply={onApplyCustomerScope}
        onDraftCustomerIdChange={onDraftCustomerIdChange}
      />

      <div className="flex flex-wrap gap-2">
        <Button onClick={() => setStatementOpen(true)} type="button" variant="outline">
          Load billing statement
        </Button>
        <Button onClick={() => setPaymentOpen(true)} type="button">
          Create payment intent
        </Button>
        <Button onClick={() => setApiKeyOpen(true)} type="button">
          Mint API key
        </Button>
        <Button onClick={() => setPauseOpen(true)} type="button" variant="outline">
          Pause or resume campaign
        </Button>
      </div>

      <Dialog onOpenChange={setStatementOpen} open={statementOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Billing statement</DialogTitle>
          </DialogHeader>
          <div className="grid gap-2">
            <Label htmlFor="selfserve-statement-month">Month (YYYY-MM)</Label>
            <Input
              id="selfserve-statement-month"
              placeholder="2026-08"
              value={draftStatementMonth}
              onChange={(event) => onDraftStatementMonthChange(event.target.value)}
            />
          </div>
          {statementError ? portalsPanelError(statementError, 'Could not load statement') : null}
          <DialogFooter>
            <Button disabled={fetchingStatement} onClick={onLoadStatement} type="button">
              {fetchingStatement ? 'Loading...' : 'Load statement'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog onOpenChange={setPaymentOpen} open={paymentOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Payment intent</DialogTitle>
          </DialogHeader>
          <div className="grid gap-2">
            <Label htmlFor="selfserve-amount-micro">Amount (micro)</Label>
            <Input
              id="selfserve-amount-micro"
              value={draftPaymentAmountMicro}
              onChange={(event) => onDraftPaymentAmountMicroChange(event.target.value)}
            />
          </div>
          <DialogFooter>
            <Button disabled={acting} onClick={onCreatePaymentIntent} type="button">
              {acting ? 'Creating...' : 'Create intent'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog onOpenChange={setApiKeyOpen} open={apiKeyOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Mint API key</DialogTitle>
          </DialogHeader>
          <div className="grid gap-2">
            <Label htmlFor="selfserve-api-key-name">Key name</Label>
            <Input
              id="selfserve-api-key-name"
              value={draftApiKeyName}
              onChange={(event) => onDraftApiKeyNameChange(event.target.value)}
            />
          </div>
          <DialogFooter>
            <Button disabled={acting} onClick={onCreateApiKey} type="button">
              {acting ? 'Minting...' : 'Mint key'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog onOpenChange={setPauseOpen} open={pauseOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Campaign pause / resume</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="selfserve-campaign-id">Campaign ID</Label>
              <Input
                id="selfserve-campaign-id"
                value={draftPauseCampaignId}
                onChange={(event) => onDraftPauseCampaignIdChange(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="selfserve-pause-reason">Reason</Label>
              <Input
                id="selfserve-pause-reason"
                value={draftPauseReason}
                onChange={(event) => onDraftPauseReasonChange(event.target.value)}
              />
            </div>
          </div>
          <DialogFooter className="gap-2 sm:justify-end">
            <Button disabled={acting} onClick={onPauseCampaign} type="button" variant="outline">
              Pause
            </Button>
            <Button disabled={acting} onClick={onResumeCampaign} type="button">
              Resume
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {statement ? (
        <section className="ui-filter-panel gap-2">
          <h2 className="text-base font-semibold">Billing statement</h2>
          <dl className="grid gap-1 text-sm">
            <div>
              <dt className="text-muted-foreground">Opening balance (micro)</dt>
              <dd>{statement.opening_balance_micro ?? ''}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Closing balance (micro)</dt>
              <dd>{statement.closing_balance_micro ?? ''}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Lines</dt>
              <dd>{statement.lines?.length ?? 0}</dd>
            </div>
          </dl>
        </section>
      ) : null}

      {paymentResult ? (
        <section className="ui-filter-panel gap-2">
          <h2 className="text-base font-semibold">Payment intent</h2>
          <dl className="grid gap-1 text-sm">
            <div>
              <dt className="text-muted-foreground">Intent</dt>
              <dd>{paymentResult.intent_id}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Status</dt>
              <dd>{paymentResult.status}</dd>
            </div>
            {paymentResult.checkout_url ? (
              <div>
                <dt className="text-muted-foreground">Checkout</dt>
                <dd>
                  <a className="underline" href={paymentResult.checkout_url} rel="noreferrer" target="_blank">
                    Open checkout
                  </a>
                </dd>
              </div>
            ) : null}
          </dl>
        </section>
      ) : null}

      {apiKeyResult ? (
        <section className="ui-filter-panel gap-2">
          <h2 className="text-base font-semibold">API key</h2>
          <p className="font-mono text-xs break-all">
            raw_key (shown once): {apiKeyResult.raw_key}
          </p>
        </section>
      ) : null}

      {actionMessage ? <p className="text-sm text-muted-foreground">{actionMessage}</p> : null}

      <section className="grid gap-2">
        <h2 className="text-base font-semibold">Invoices</h2>
        {fetchingInvoices && !hasInvoicesSnapshot && !invoicesError ? (
          <PageSkeleton />
        ) : invoicesError && !hasInvoicesSnapshot ? (
          portalsPanelError(invoicesError, 'Could not load self-serve invoices')
        ) : invoices.length === 0 ? (
          <EmptyState title="No invoices" description="Self-serve invoice list is empty." />
        ) : (
          <div className="ui-table-frame">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Period</TableHead>
                  <TableHead>Total (micro)</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {invoices.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell className="font-mono text-xs">{row.id}</TableCell>
                    <TableCell>{row.status ?? ''}</TableCell>
                    <TableCell>{row.billing_month}</TableCell>
                    <TableCell>{displayMicro(row.total_micro, row.total_micro_display)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </section>

      {actionError ? portalsPanelError(actionError, 'Self-serve action failed') : null}
    </PageChrome>
  );
}
