import { apiJson } from './client.js';
import type { CampaignForecast, CampaignForecastRequest } from './types.js';

export async function forecastCampaign(
  body: CampaignForecastRequest,
  signal?: AbortSignal,
): Promise<CampaignForecast> {
  return apiJson<CampaignForecast>('/api/v1/forecast/campaign', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}
