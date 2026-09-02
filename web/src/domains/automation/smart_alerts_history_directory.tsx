import { PageChrome } from '@/shell/page_chrome';
import { CustomerScopeBar } from '@/shell/customer_scope_bar';
import { EmptyState } from '@/shell/empty_state';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { SmartAlertEvent } from '@/api/types';
import { AutomationNav, automationPanelError } from '@/domains/automation/automation_nav';
import { displayTimestamp } from '@/lib/display';

export type SmartAlertsHistoryDirectoryProps = {
  items: SmartAlertEvent[];
  appliedCustomerId: string;
  draftCustomerId: string;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  ackingEventId: string | undefined;
  ackError: Error | undefined;
  onDraftCustomerIdChange: (value: string) => void;
  onApplyCustomerScope: () => void;
  onAck: (eventId: string) => void;
};

export function SmartAlertsHistoryDirectory({
  items,
  appliedCustomerId,
  draftCustomerId,
  fetching,
  error,
  hasSnapshot,
  ackingEventId,
  ackError,
  onDraftCustomerIdChange,
  onApplyCustomerScope,
  onAck,
}: SmartAlertsHistoryDirectoryProps) {
  if (!appliedCustomerId) {
    return (
      <PageChrome title="Smart alert history">
        <AutomationNav />
        <CustomerScopeBar
          appliedCustomerId={appliedCustomerId}
          draftCustomerId={draftCustomerId}
          onApply={onApplyCustomerScope}
          onDraftCustomerIdChange={onDraftCustomerIdChange}
        />
        <EmptyState
          title="Customer required"
          description="Apply a customer ID to list smart alert history."
        />
      </PageChrome>
    );
  }

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Smart alert history">
        <AutomationNav />
        <CustomerScopeBar
          appliedCustomerId={appliedCustomerId}
          draftCustomerId={draftCustomerId}
          onApply={onApplyCustomerScope}
          onDraftCustomerIdChange={onDraftCustomerIdChange}
        />
        {automationPanelError(error, 'Could not load smart alert history')}
      </PageChrome>
    );
  }

  return (
    <PageChrome title="Smart alert history">
      <AutomationNav />

      <CustomerScopeBar
        appliedCustomerId={appliedCustomerId}
        draftCustomerId={draftCustomerId}
        onApply={onApplyCustomerScope}
        onDraftCustomerIdChange={onDraftCustomerIdChange}
      />

      {items.length === 0 ? (
        <EmptyState title="No alert events" description="No smart alert events for this customer." />
      ) : (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Metric</TableHead>
                <TableHead>Observed</TableHead>
                <TableHead>Threshold</TableHead>
                <TableHead>Fired</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Ack</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((row) => (
                <TableRow key={row.id}>
                  <TableCell>{row.metric}</TableCell>
                  <TableCell>{row.observed_value}</TableCell>
                  <TableCell>{row.threshold}</TableCell>
                  <TableCell>{displayTimestamp(row.fired_at)}</TableCell>
                  <TableCell>
                    {row.acked_at ? (
                      <Badge variant="outline">acked</Badge>
                    ) : (
                      <Badge variant="secondary">{row.webhook_status}</Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    {row.acked_at ? null : (
                      <Button
                        disabled={ackingEventId === row.id}
                        onClick={() => onAck(row.id)}
                       
                        type="button"
                        variant="outline"
                      >
                        Ack
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {ackError ? automationPanelError(ackError, 'Ack failed') : null}
      {error && hasSnapshot ? automationPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
