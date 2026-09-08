import { Link } from 'react-router-dom';

import { SecondaryActionButton } from '@/shell/action_buttons';
import {
  DirectoryFilterForm,
  FilterField,
  FilterPanel,
  INLINE_FILTER_ACTION_GRID_CLASS,
} from '@/shell/filter_panel';
import { PageChrome } from '@/shell/page_chrome';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
  directoryTableRevalidatingClass,
} from '@/shell/directory_table';
import type { FraudIntegration } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { TableHost } from '@/shell/ui_bands';

export type FraudIntegrationsProps = {
  items?: FraudIntegration[];
  customerId: string;
  draftCustomerId: string;
  fetching: boolean;
  listRevalidating?: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  onDraftCustomerIdChange: (value: string) => void;
  onApplyCustomer: () => void;
};

function statusBadgeVariant(
  status: string | undefined
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
  listRevalidating = false,
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
    <PageChrome
      controlPanel={
        <div className="grid gap-3">
          <Link className="text-sm text-muted-foreground hover:underline" to="/fraud">
            Back to fraud hub
          </Link>
          <FilterPanel>
            <DirectoryFilterForm
              className={INLINE_FILTER_ACTION_GRID_CLASS}
              onSubmit={(event) => {
                event.preventDefault();
                onApplyCustomer();
              }}
            >
              <FilterField htmlFor="fraud-customer-id" label="Customer ID">
                <Input
                  id="fraud-customer-id"
                  value={draftCustomerId}
                  onChange={(event) => onDraftCustomerIdChange(event.target.value)}
                />
              </FilterField>
              <SecondaryActionButton disabled={fetching || !draftCustomerId.trim()} type="submit">
                Load
              </SecondaryActionButton>
            </DirectoryFilterForm>
          </FilterPanel>
        </div>
      }
      title="Fraud integrations"
    >
      {!customerId ? (
        <EmptyState
          title="Customer required"
          description="Enter a customer ID to list third-party fraud integration health."
        />
      ) : (items ?? []).length === 0 ? (
        <EmptyState
          title="No integrations"
          description="No fraud integrations are configured for this customer."
        />
      ) : (
        <TableHost className="w-full">
          <DirectoryTable
            className={directoryTableRevalidatingClass(listRevalidating)}
            horizontalScroll
            nested
          >
            <TableHeader>
              <TableRow>
                <DirectoryTableHead>Campaign</DirectoryTableHead>
                <DirectoryTableHead>Name</DirectoryTableHead>
                <DirectoryTableHead>Provider</DirectoryTableHead>
                <DirectoryTableHead>Configured</DirectoryTableHead>
                <DirectoryTableHead>Health</DirectoryTableHead>
                <DirectoryTableHead className="text-right">DLQ</DirectoryTableHead>
                <DirectoryTableHead>Last success</DirectoryTableHead>
                <DirectoryTableHead>Error</DirectoryTableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(items ?? []).map((row) => (
                <TableRow key={`${row.campaign_id}-${row.provider ?? row.name}`}>
                  <TableCell className="text-xs">{row.campaign_id}</TableCell>
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
                  <TableCell>{displayTimestamp(row.last_success_at)}</TableCell>
                  <TableCell className="whitespace-nowrap text-muted-foreground">
                    {row.last_error ?? ''}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </DirectoryTable>
        </TableHost>
      )}

      {error && hasSnapshot ? <ErrorBlock title="Refresh failed" message={error.message} /> : null}
    </PageChrome>
  );
}
