import { api } from './api_client.js';

export type FraudIntegrationHealthStatus = 'configured' | 'failing' | 'idle' | 'unconfigured';

export type FraudIntegrationRow = {
  campaign_id: string;
  name: string;
  provider?: string;
  configured: boolean;
  health_status: FraudIntegrationHealthStatus;
  last_success_at?: string;
  dlq_count: number;
  last_error?: string;
};

/**
 * List postback/CAPI integration health for a customer scope.
 */
export async function fetchFraudIntegrations(customerId: string): Promise<FraudIntegrationRow[]> {
  const qs = new URLSearchParams({ customer_id: customerId });
  const res = await api<FraudIntegrationRow[]>(`/api/v1/fraud/integrations?${qs.toString()}`);
  return Array.isArray(res.data) ? res.data : [];
}

/**
 * Human-readable label for integration health status badges.
 */
export function fraudIntegrationStatusLabel(status: FraudIntegrationHealthStatus): string {
  switch (status) {
    case 'configured':
      return 'Configured';
    case 'failing':
      return 'Failing';
    case 'idle':
      return 'Idle';
    case 'unconfigured':
      return 'Not configured';
    default:
      return status;
  }
}

/**
 * Map integration health status to StatusBadge status.
 */
export function fraudIntegrationBadgeStatus(
  status: FraudIntegrationHealthStatus
): 'ok' | 'warning' | 'failed' | 'pending' {
  switch (status) {
    case 'configured':
      return 'ok';
    case 'idle':
      return 'warning';
    case 'failing':
      return 'failed';
    default:
      return 'pending';
  }
}
