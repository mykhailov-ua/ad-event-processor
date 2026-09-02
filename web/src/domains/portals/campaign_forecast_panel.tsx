import { PageChrome } from '@/shell/page_chrome';
import { CustomerScopeBar } from '@/shell/customer_scope_bar';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import type { CampaignForecast } from '@/api/types';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import { PortalsNav, portalsPanelError } from '@/domains/portals/portals_nav';

export type CampaignForecastPanelProps = {
  appliedCustomerId: string;
  draftCustomerId: string;
  draftBudgetLimitMicro: string;
  draftStartAt: string;
  draftEndAt: string;
  forecast: CampaignForecast | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  onDraftCustomerIdChange: (value: string) => void;
  onApplyCustomerScope: () => void;
  onDraftBudgetLimitMicroChange: (value: string) => void;
  onDraftStartAtChange: (value: string) => void;
  onDraftEndAtChange: (value: string) => void;
  onRunForecast: () => void;
};

export function CampaignForecastPanel({
  appliedCustomerId,
  draftCustomerId,
  draftBudgetLimitMicro,
  draftStartAt,
  draftEndAt,
  forecast,
  fetching,
  error,
  hasSnapshot,
  onDraftCustomerIdChange,
  onApplyCustomerScope,
  onDraftBudgetLimitMicroChange,
  onDraftStartAtChange,
  onDraftEndAtChange,
  onRunForecast,
}: CampaignForecastPanelProps) {
  const canRun =
    Boolean(appliedCustomerId) &&
    Boolean(draftStartAt.trim()) &&
    Boolean(draftEndAt.trim()) &&
    Boolean(draftBudgetLimitMicro.trim());

  return (
    <PageChrome title="Campaign forecast">
      <PortalsNav />

      <CustomerScopeBar
        appliedCustomerId={appliedCustomerId}
        draftCustomerId={draftCustomerId}
        onApply={onApplyCustomerScope}
        onDraftCustomerIdChange={onDraftCustomerIdChange}
      />

      <section className="ui-filter-panel max-w-xl">
        <div className="grid gap-2">
          <Label htmlFor="forecast-budget-micro">Budget limit (micro)</Label>
          <Input
            id="forecast-budget-micro"
            value={draftBudgetLimitMicro}
            onChange={(event) => onDraftBudgetLimitMicroChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="forecast-start-at">Start (ISO 8601)</Label>
          <Input
            id="forecast-start-at"
            value={draftStartAt}
            onChange={(event) => onDraftStartAtChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="forecast-end-at">End (ISO 8601)</Label>
          <Input
            id="forecast-end-at"
            value={draftEndAt}
            onChange={(event) => onDraftEndAtChange(event.target.value)}
          />
        </div>
        <Button disabled={fetching || !canRun} onClick={onRunForecast} type="button">
          Run forecast
        </Button>
      </section>

      {fetching && !hasSnapshot && !error ? (
        <PageSkeleton />
      ) : error && !hasSnapshot ? (
        portalsPanelError(error, 'Could not run campaign forecast')
      ) : forecast ? (
        <JsonDashboardView payload={forecast as unknown as Record<string, unknown>} />
      ) : null}

      {error && hasSnapshot ? portalsPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
