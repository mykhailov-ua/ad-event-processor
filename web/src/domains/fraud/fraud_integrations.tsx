import { Link } from 'react-router-dom';

import { SecondaryActionButton } from '@/components/system/action_buttons';
import { PageChrome } from '@/components/system/page_chrome';
import { EmptyState } from '@/components/system/empty_state';
import { ErrorBlock } from '@/components/system/error_block';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { Badge } from '@/components/ui/badge';
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
import type { FraudIntegration } from '@/api/types';
import { displayTimestamp } from '@/lib/display';

export type FraudIntegrationsProps = {
  items: FraudIntegration[];
  customerId: string;
  draftCustomerId: string;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  onDraftCustomerIdChange: (value: string) => void;
  onApplyCustomer: () => void;
};

function statusBadgeVariant(
  status: string | undefined,
): 'default' | 'secondary' | 'destructive' | 'outline' {
  const normalized = (status ?? '').toLowerCase();
  if (normalized === 'ok' || normalized === 'healthy') {
    return 'default';
  }
  if (normalized === 'degraded' || normalized === 'warn') {
    return 'secondary';
  }
  if (normalized === 'down' || normalized === 'error') {
    return 'destructive';
  }
  return 'outline';
}

export function FraudIntegrations({
  items,
  customerId,
  draftCustomerId,
  fetching,
  error,
  hasSnapshot,
  onDraftCustomerIdChange,
  onApplyCustomer,
}: FraudIntegrationsProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load fraud integrations" message={error.message} />;
  }

  return (
    <PageChrome title="Fraud integrations">
      <Link className="text-sm text-muted-foreground hover:underline" to="/fraud">
        Back to fraud hub
      </Link>
      <div className="grid max-w-md grid-cols-[1fr_auto] items-end gap-4">
        <div className="grid gap-2">
          <Label htmlFor="fraud-customer-id">Customer ID</Label>
          <Input
            id="fraud-customer-id"
            value={draftCustomerId}
            onChange={(event) => onDraftCustomerIdChange(event.target.value)}
          />
        </div>
        <SecondaryActionButton disabled={fetching || !draftCustomerId.trim()} onClick={onApplyCustomer} type="button">
          Load
        </SecondaryActionButton>
      </div>

      {!customerId ? (
        <EmptyState
          title="Customer required"
          description="Enter a customer ID to list third-party fraud integration health."
        />
      ) : items.length === 0 ? (
        <EmptyState
          title="No integrations"
          description="No fraud integrations are configured for this customer."
        />
      ) : (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Campaign</TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Provider</TableHead>
                <TableHead>Configured</TableHead>
                <TableHead>Health</TableHead>
                <TableHead className="text-right">DLQ</TableHead>
                <TableHead>Last success</TableHead>
                <TableHead>Error</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((row) => (
                <TableRow key={`${row.campaign_id}-${row.provider ?? row.name}`}>
                  <TableCell className="font-mono text-xs">{row.campaign_id}</TableCell>
                  <TableCell>{row.name ?? ''}</TableCell>
                  <TableCell>{row.provider ?? ''}</TableCell>
                  <TableCell>{row.configured ? 'yes' : 'no'}</TableCell>
                  <TableCell>
                    {row.health_status ? (
                      <Badge variant={statusBadgeVariant(row.health_status)}>
                        {row.health_status}
                      </Badge>
                    ) : (
                      ''
                    )}
                  </TableCell>
                  <TableCell className="text-right">{row.dlq_count ?? 0}</TableCell>
                  <TableCell>
                    {displayTimestamp(row.last_success_at)}
                  </TableCell>
                  <TableCell className="max-w-xs truncate text-muted-foreground">
                    {row.last_error ?? ''}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {error && hasSnapshot ? <ErrorBlock title="Refresh failed" message={error.message} /> : null}
    </PageChrome>
  );
}
