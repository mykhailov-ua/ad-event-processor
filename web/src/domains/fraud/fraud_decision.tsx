import { Link } from 'react-router-dom';

import { PageChrome } from '@/components/system/page_chrome';
import { ErrorBlock } from '@/components/system/error_block';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import type { FraudDecision } from '@/api/types';
import { displayTimestamp } from '@/lib/display';

export type FraudDecisionViewProps = {
  decision: FraudDecision | undefined;
  draftCustomerId: string;
  draftIpHash: string;
  draftCampaignId: string;
  draftHours: string;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  onDraftCustomerIdChange: (value: string) => void;
  onDraftIpHashChange: (value: string) => void;
  onDraftCampaignIdChange: (value: string) => void;
  onDraftHoursChange: (value: string) => void;
  onExplain: () => void;
};

export function FraudDecisionView({
  decision,
  draftCustomerId,
  draftIpHash,
  draftCampaignId,
  draftHours,
  fetching,
  error,
  hasSnapshot,
  onDraftCustomerIdChange,
  onDraftIpHashChange,
  onDraftCampaignIdChange,
  onDraftHoursChange,
  onExplain,
}: FraudDecisionViewProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not explain fraud decision" message={error.message} />;
  }

  const features = decision?.features ? Object.entries(decision.features) : [];

  return (
    <PageChrome title="Fraud decision explain">
      <Link className="text-sm text-muted-foreground hover:underline" to="/fraud">
        Back to fraud hub
      </Link>

      <div className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] items-end gap-4">
        <div className="grid gap-2 md:col-span-2">
          <Label htmlFor="decision-customer-id">Customer ID</Label>
          <Input
            id="decision-customer-id"
            value={draftCustomerId}
            onChange={(event) => onDraftCustomerIdChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2 md:col-span-2">
          <Label htmlFor="decision-ip-hash">IP hash</Label>
          <Input
            id="decision-ip-hash"
            value={draftIpHash}
            onChange={(event) => onDraftIpHashChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="decision-campaign-id">Campaign ID</Label>
          <Input
            id="decision-campaign-id"
            value={draftCampaignId}
            onChange={(event) => onDraftCampaignIdChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="decision-hours">Hours</Label>
          <Input
            id="decision-hours"
            inputMode="numeric"
            value={draftHours}
            onChange={(event) => onDraftHoursChange(event.target.value)}
          />
        </div>
        <Button
          disabled={fetching || !draftCustomerId.trim() || !draftIpHash.trim()}
          onClick={onExplain}
          type="button"
        >
          Explain
        </Button>
      </div>

      {decision ? (
        <div className="grid gap-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="flex flex-wrap items-center gap-2 text-base">
                Tier
                {decision.tier ? <Badge>{decision.tier}</Badge> : null}
                {decision.score != null ? <span>score {decision.score}</span> : null}
              </CardTitle>
            </CardHeader>
            <CardContent className="grid gap-2 text-sm">
              <div>IP hash: {decision.ip_hash ?? ''}</div>
              {decision.campaign_id ? <div>Campaign: {decision.campaign_id}</div> : null}
              <div>
                Evaluated:{' '}
                {displayTimestamp(decision.evaluated_at, decision.evaluated_at_display)}
              </div>
              {decision.disclaimer ? (
                <p className="text-muted-foreground">{decision.disclaimer}</p>
              ) : null}
              {decision.model_name ? <div>Model: {decision.model_name}</div> : null}
              {decision.ml_probability != null ? (
                <div>ML probability: {decision.ml_probability.toFixed(4)}</div>
              ) : null}
              {decision.adjusted_probability != null ? (
                <div>Adjusted probability: {decision.adjusted_probability.toFixed(4)}</div>
              ) : null}
            </CardContent>
          </Card>

          {features.length > 0 ? (
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-base">Features</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-1 text-sm">
                {features.map(([key, value]) => (
                  <div className="flex justify-between gap-4" key={key}>
                    <span className="text-muted-foreground">{key}</span>
                    <span>{value}</span>
                  </div>
                ))}
              </CardContent>
            </Card>
          ) : null}
        </div>
      ) : null}

      {error && hasSnapshot ? <ErrorBlock title="Refresh failed" message={error.message} /> : null}
    </PageChrome>
  );
}
