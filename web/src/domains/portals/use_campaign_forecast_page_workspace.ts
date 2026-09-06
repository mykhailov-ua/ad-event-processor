// L3 campaign forecast tool: lazy POST forecast via loadToken + skipLazyFetch until Run.
import { useState } from 'react';

import { forecastCampaign } from '@/api/forecast_api';
import type { CampaignForecastRequest } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { useCoalescedCallback } from '@/hooks/use_coalesced_callback';
import { useCustomerScope } from '@/hooks/use_customer_scope';

function skipLazyFetch(): Promise<never> {
  return Promise.reject(new DOMException('Skipped', 'AbortError'));
}

export function useCampaignForecastPageWorkspace() {
  const { appliedCustomerId, draftCustomerId, setDraftCustomerId, applyCustomerScope } =
    useCustomerScope();

  const [draftBudgetLimitMicro, setDraftBudgetLimitMicro] = useState('');
  const [draftStartAt, setDraftStartAt] = useState('');
  const [draftEndAt, setDraftEndAt] = useState('');
  const [forecastLoadToken, setForecastLoadToken] = useState(0);
  const [forecastRequest, setForecastRequest] = useState<CampaignForecastRequest | undefined>();
  const [validationError, setValidationError] = useState<Error | undefined>();

  const forecastResource = useResource(
    (signal) => {
      if (forecastLoadToken === 0 || !forecastRequest) {
        return skipLazyFetch();
      }
      return forecastCampaign(forecastRequest, signal);
    },
    [forecastLoadToken, forecastRequest]
  );

  const onApplyCustomerScope = useCoalescedCallback(applyCustomerScope, {});

  const onRunForecast = useCoalescedCallback(
    () => {
      const budgetLimitMicro = Number.parseInt(draftBudgetLimitMicro.trim(), 10);
      const startAt = draftStartAt.trim();
      const endAt = draftEndAt.trim();
      if (!appliedCustomerId || !Number.isFinite(budgetLimitMicro) || !startAt || !endAt) {
        setValidationError(new Error('Customer, budget, start_at, and end_at are required'));
        return;
      }
      setValidationError(undefined);
      setForecastRequest({
        customer_id: appliedCustomerId,
        budget_limit_micro: budgetLimitMicro,
        start_at: startAt,
        end_at: endAt,
      });
      setForecastLoadToken((value) => value + 1);
    },
    {
      inFlightGuard: true,
      inFlight: forecastResource.fetching,
    }
  );

  return {
    appliedCustomerId,
    draftCustomerId,
    draftBudgetLimitMicro,
    draftStartAt,
    draftEndAt,
    forecast: forecastResource.data,
    fetching: forecastResource.fetching,
    error: validationError ?? forecastResource.error,
    hasSnapshot: forecastResource.data != null,
    onDraftCustomerIdChange: setDraftCustomerId,
    onApplyCustomerScope: onApplyCustomerScope,
    onDraftBudgetLimitMicroChange: setDraftBudgetLimitMicro,
    onDraftStartAtChange: setDraftStartAt,
    onDraftEndAtChange: setDraftEndAt,
    onRunForecast,
  };
}
