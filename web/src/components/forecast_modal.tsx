import { useCallback, useEffect, useRef, useState } from 'react';
import { to } from '../lib/to.js';
import { api, ApiError } from '../helpers/api_client.js';
import { Modal } from './modal.js';
import { CampaignSpendCurveChart } from './campaign_hourly_chart.js';
import type { SpendCurvePoint } from '../charts/campaign_chart_types.js';

export type ForecastModalOpts = {
  campaignId: string;
  customerId?: string;
  budgetMicro?: number;
  startAt: string;
  endAt: string;
};

type ForecastAdvisory = {
  message?: string;
};

type ForecastPayload = {
  impressions_p50?: number;
  impressions_p90?: number;
  low_confidence?: boolean;
  advisory?: ForecastAdvisory;
  spend_curve?: SpendCurvePoint[];
};

export type ForecastModalProps = {
  open: boolean;
  opts: ForecastModalOpts | null;
  onClose: () => void;
};

export function ForecastModal({ open, opts, onClose }: ForecastModalProps) {
  const [phase, setPhase] = useState<'idle' | 'loading' | 'retry' | 'error' | 'ready'>('idle');
  const [countdown, setCountdown] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [forecast, setForecast] = useState<ForecastPayload | null>(null);
  const retryTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const optsRef = useRef(opts);
  optsRef.current = opts;

  const clearRetryTimer = useCallback(() => {
    if (retryTimerRef.current) {
      clearInterval(retryTimerRef.current);
      retryTimerRef.current = null;
    }
  }, []);

  const fetchForecast = useCallback(async () => {
    const current = optsRef.current;
    if (!current) return;

    setPhase('loading');
    setError(null);
    setForecast(null);

    const payload: Record<string, unknown> = {
      budget_limit_micro: current.budgetMicro ?? 0,
      start_at: current.startAt,
      end_at: current.endAt,
      pacing_mode: 'even',
      timezone: 'UTC',
    };
    if (current.customerId) payload.customer_id = current.customerId;

    const [res, err] = await to(
      api('/api/v1/forecast/campaign', {
        method: 'POST',
        body: JSON.stringify(payload),
      })
    );

    if (err) {
      const retryAfter =
        err instanceof ApiError
          ? Number(
              err.responseHeaders?.get('Retry-After') ??
                (err as Error & { retryAfter?: number }).retryAfter ??
                0
            )
          : Number((err as Error & { retryAfter?: number }).retryAfter ?? 0);
      const status =
        err instanceof ApiError ? err.status : (err as Error & { status?: number }).status;
      if (status === 503 && retryAfter > 0) {
        clearRetryTimer();
        let remaining = retryAfter;
        setCountdown(remaining);
        setPhase('retry');
        retryTimerRef.current = setInterval(() => {
          remaining -= 1;
          setCountdown(remaining);
          if (remaining <= 0) {
            clearRetryTimer();
            void fetchForecast();
          }
        }, 1000);
        return;
      }
      setError(err.message ?? 'Forecast failed');
      setPhase('error');
      return;
    }

    setForecast((res?.data as ForecastPayload | null) ?? null);
    setPhase('ready');
  }, [clearRetryTimer]);

  useEffect(() => {
    if (!open || !opts) {
      clearRetryTimer();
      setPhase('idle');
      setCountdown(0);
      setError(null);
      setForecast(null);
      return undefined;
    }
    void fetchForecast();
    return () => clearRetryTimer();
  }, [open, opts, fetchForecast, clearRetryTimer]);

  const handleClose = () => {
    clearRetryTimer();
    onClose();
  };

  let statusMessage: string | null = null;
  if (phase === 'loading') statusMessage = 'Loading forecast...';
  else if (phase === 'retry') statusMessage = `Service busy - retry in ${countdown}s`;
  else if (phase === 'error') statusMessage = error;

  const curve = forecast?.spend_curve ?? [];

  return (
    <Modal
      open={open}
      title="Campaign forecast"
      onClose={handleClose}
      testId="campaign-forecast-modal"
    >
      {statusMessage ? <p>{statusMessage}</p> : null}
      {phase === 'ready' && forecast ? (
        <div className="stack">
          <dl>
            <dt>Impressions P50</dt>
            <dd className="font-mono">{String(forecast.impressions_p50 ?? '-')}</dd>
            <dt>Impressions P90</dt>
            <dd className="font-mono">{String(forecast.impressions_p90 ?? '-')}</dd>
            <dt>Low confidence</dt>
            <dd>{forecast.low_confidence ? 'Yes' : 'No'}</dd>
          </dl>
          {forecast.advisory?.message ? (
            <p className="text-muted">{forecast.advisory.message}</p>
          ) : null}
          {curve.length > 0 ? (
            <>
              <h3 className="subsection-title">Projected delivery</h3>
              <CampaignSpendCurveChart curve={curve} field="impressions" />
            </>
          ) : null}
        </div>
      ) : null}
      {phase === 'ready' && !forecast ? <p>No forecast data.</p> : null}
    </Modal>
  );
}

export function useForecastModal() {
  const [opts, setOpts] = useState<ForecastModalOpts | null>(null);
  return {
    forecastOpen: opts != null,
    forecastOpts: opts,
    openForecast: setOpts,
    closeForecast: () => setOpts(null),
  };
}
