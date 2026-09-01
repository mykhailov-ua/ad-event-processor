import { useCallback, useState } from 'react';

import { forecastCampaign } from '@/api/forecast_api';
import type { CampaignForecast } from '@/api/types';
import { CampaignForecastPanel } from '@/domains/portals/campaign_forecast_panel';
import { useCustomerScope } from '@/hooks/use_customer_scope';

export function CampaignForecastPage() {
  const {
    appliedCustomerId,
    draftCustomerId,
    setDraftCustomerId,
    applyCustomerScope,
  } = useCustomerScope();

  const [draftBudgetLimitMicro, setDraftBudgetLimitMicro] = useState('');
  const [draftStartAt, setDraftStartAt] = useState('');
  const [draftEndAt, setDraftEndAt] = useState('');
  const [forecast, setForecast] = useState<CampaignForecast | undefined>(undefined);
  const [fetching, setFetching] = useState(false);
  const [error, setError] = useState<Error | undefined>(undefined);
  const [hasSnapshot, setHasSnapshot] = useState(false);

  const onRunForecast = useCallback(() => {
    const budgetLimitMicro = Number.parseInt(draftBudgetLimitMicro.trim(), 10);
    const startAt = draftStartAt.trim();
    const endAt = draftEndAt.trim();
    if (!appliedCustomerId || !Number.isFinite(budgetLimitMicro) || !startAt || !endAt) {
      setError(new Error('Customer, budget, start_at, and end_at are required'));
      return;
    }

    setFetching(true);
    setError(undefined);
    void forecastCampaign({
      customer_id: appliedCustomerId,
      budget_limit_micro: budgetLimitMicro,
      start_at: startAt,
      end_at: endAt,
    })
      .then((result) => {
        setForecast(result);
        setHasSnapshot(true);
      })
      .catch((err: unknown) => {
        setError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setFetching(false);
      });
  }, [appliedCustomerId, draftBudgetLimitMicro, draftEndAt, draftStartAt]);

  return (
    <CampaignForecastPanel
      appliedCustomerId={appliedCustomerId}
      draftCustomerId={draftCustomerId}
      draftBudgetLimitMicro={draftBudgetLimitMicro}
      draftStartAt={draftStartAt}
      draftEndAt={draftEndAt}
      forecast={forecast}
      fetching={fetching}
      error={error}
      hasSnapshot={hasSnapshot}
      onDraftCustomerIdChange={setDraftCustomerId}
      onApplyCustomerScope={applyCustomerScope}
      onDraftBudgetLimitMicroChange={setDraftBudgetLimitMicro}
      onDraftStartAtChange={setDraftStartAt}
      onDraftEndAtChange={setDraftEndAt}
      onRunForecast={onRunForecast}
    />
  );
}
