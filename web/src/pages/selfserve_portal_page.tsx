import { useCallback, useMemo, useState } from 'react';
import { toast } from 'sonner';

import {
  createSelfServeApiKey,
  createSelfServePaymentIntent,
  getSelfServeBillingStatement,
  listSelfServeInvoices,
  pauseSelfServeCampaign,
  resumeSelfServeCampaign,
} from '@/api/selfserve_api';
import type { APIKeyCreatedResponse, BillingStatement, PaymentIntentCreatedResponse } from '@/api/types';
import { SelfServePortal } from '@/domains/portals/selfserve_portal';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useResource } from '@/api/use_resource';

export function SelfServePortalPage() {
  const {
    appliedCustomerId,
    draftCustomerId,
    setDraftCustomerId,
    applyCustomerScope,
  } = useCustomerScope();

  const shouldFetch = Boolean(appliedCustomerId);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve({ invoices: [], total: 0 });
      }
      return listSelfServeInvoices({ customer_id: appliedCustomerId }, signal);
    },
    [appliedCustomerId, shouldFetch],
  );

  const [draftPaymentAmountMicro, setDraftPaymentAmountMicro] = useState('');
  const [draftApiKeyName, setDraftApiKeyName] = useState('');
  const [draftPauseCampaignId, setDraftPauseCampaignId] = useState('');
  const [draftPauseReason, setDraftPauseReason] = useState('');
  const [acting, setActing] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>(undefined);
  const [paymentResult, setPaymentResult] = useState<PaymentIntentCreatedResponse | undefined>(
    undefined,
  );
  const [apiKeyResult, setApiKeyResult] = useState<APIKeyCreatedResponse | undefined>(undefined);
  const [actionMessage, setActionMessage] = useState<string | undefined>(undefined);
  const [draftStatementMonth, setDraftStatementMonth] = useState('');
  const [statement, setStatement] = useState<BillingStatement | undefined>();
  const [fetchingStatement, setFetchingStatement] = useState(false);
  const [statementError, setStatementError] = useState<Error | undefined>(undefined);

  const invoices = useMemo(() => data?.invoices ?? [], [data]);

  const runAction = useCallback(async (action: () => Promise<void>) => {
    setActing(true);
    setActionError(undefined);
    setActionMessage(undefined);
    try {
      await action();
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setActing(false);
    }
  }, []);

  const onCreatePaymentIntent = useCallback(() => {
    const amountMicro = Number.parseInt(draftPaymentAmountMicro.trim(), 10);
    if (!appliedCustomerId || !Number.isFinite(amountMicro) || amountMicro <= 0) {
      setActionError(new Error('Customer and positive amount_micro are required'));
      return;
    }
    void runAction(async () => {
      const result = await createSelfServePaymentIntent({
        customer_id: appliedCustomerId,
        amount_micro: amountMicro,
        currency: 'USD',
      });
      setPaymentResult(result);
      toast.success('Payment intent created');
    });
  }, [appliedCustomerId, draftPaymentAmountMicro, runAction]);

  const onCreateApiKey = useCallback(() => {
    const name = draftApiKeyName.trim();
    if (!name) {
      setActionError(new Error('Key name is required'));
      return;
    }
    void runAction(async () => {
      const result = await createSelfServeApiKey({ name });
      setApiKeyResult(result);
      toast.success('API key minted');
    });
  }, [draftApiKeyName, runAction]);

  const onPauseCampaign = useCallback(() => {
    const campaignId = draftPauseCampaignId.trim();
    if (!campaignId) {
      setActionError(new Error('Campaign ID is required'));
      return;
    }
    void runAction(async () => {
      await pauseSelfServeCampaign(campaignId, {
        reason: draftPauseReason.trim() || undefined,
      });
      setActionMessage(`Pause accepted for ${campaignId}`);
      toast.success('Campaign pause accepted');
    });
  }, [draftPauseCampaignId, draftPauseReason, runAction]);

  const onResumeCampaign = useCallback(() => {
    const campaignId = draftPauseCampaignId.trim();
    if (!campaignId) {
      setActionError(new Error('Campaign ID is required'));
      return;
    }
    void runAction(async () => {
      await resumeSelfServeCampaign(campaignId, {
        reason: draftPauseReason.trim() || undefined,
      });
      setActionMessage(`Resume accepted for ${campaignId}`);
      toast.success('Campaign resume accepted');
    });
  }, [draftPauseCampaignId, draftPauseReason, runAction]);

  const onLoadStatement = useCallback(() => {
    if (!appliedCustomerId) {
      setStatementError(new Error('Customer ID is required'));
      return;
    }
    setFetchingStatement(true);
    setStatementError(undefined);
    void getSelfServeBillingStatement({
      customer_id: appliedCustomerId,
      month: draftStatementMonth.trim() || undefined,
    })
      .then((result) => {
        setStatement(result);
        toast.success('Billing statement loaded');
      })
      .catch((err: unknown) => {
        setStatementError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setFetchingStatement(false);
      });
  }, [appliedCustomerId, draftStatementMonth]);

  return (
    <SelfServePortal
      invoices={invoices}
      appliedCustomerId={appliedCustomerId}
      draftCustomerId={draftCustomerId}
      fetchingInvoices={fetching}
      invoicesError={error}
      hasInvoicesSnapshot={!shouldFetch || data != null}
      draftPaymentAmountMicro={draftPaymentAmountMicro}
      draftApiKeyName={draftApiKeyName}
      draftPauseCampaignId={draftPauseCampaignId}
      draftPauseReason={draftPauseReason}
      acting={acting}
      actionError={actionError}
      paymentResult={paymentResult}
      apiKeyResult={apiKeyResult}
      actionMessage={actionMessage}
      onDraftCustomerIdChange={setDraftCustomerId}
      onApplyCustomerScope={applyCustomerScope}
      onDraftPaymentAmountMicroChange={setDraftPaymentAmountMicro}
      onDraftApiKeyNameChange={setDraftApiKeyName}
      onDraftPauseCampaignIdChange={setDraftPauseCampaignId}
      onDraftPauseReasonChange={setDraftPauseReason}
      onCreatePaymentIntent={onCreatePaymentIntent}
      onCreateApiKey={onCreateApiKey}
      onPauseCampaign={onPauseCampaign}
      onResumeCampaign={onResumeCampaign}
      draftStatementMonth={draftStatementMonth}
      statement={statement}
      fetchingStatement={fetchingStatement}
      statementError={statementError}
      onDraftStatementMonthChange={setDraftStatementMonth}
      onLoadStatement={onLoadStatement}
    />
  );
}
