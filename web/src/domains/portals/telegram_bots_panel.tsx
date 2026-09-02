import { PageChrome } from '@/shell/page_chrome';
import { EmptyState } from '@/shell/empty_state';
import { PageSkeleton } from '@/shell/page_skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { TelegramBot } from '@/api/types';
import { PortalsNav, portalsPanelError } from '@/domains/portals/portals_nav';
import { displayTimestamp } from '@/lib/display';

export type TelegramBotsPanelProps = {
  bots: TelegramBot[];
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
};

export function TelegramBotsPanel({ bots, fetching, error, hasSnapshot }: TelegramBotsPanelProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Telegram bots">
        <PortalsNav />
        {portalsPanelError(error, 'Could not load Telegram bots')}
      </PageChrome>
    );
  }

  return (
    <PageChrome title="Telegram bots">
      <PortalsNav />

      {bots.length === 0 ? (
        <EmptyState title="No bots" description="No Telegram Mini App bots are configured." />
      ) : (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Bot ID</TableHead>
                <TableHead>Campaign</TableHead>
                <TableHead>Webhook</TableHead>
                <TableHead>Mini App</TableHead>
                <TableHead>Updated</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {bots.map((row) => (
                <TableRow key={String(row.bot_id ?? row.campaign_id ?? row.webhook_url)}>
                  <TableCell>{row.bot_id ?? ''}</TableCell>
                  <TableCell className="font-mono text-xs">{row.campaign_id ?? ''}</TableCell>
                  <TableCell className="max-w-xs truncate font-mono text-xs">
                    {row.webhook_url ?? ''}
                  </TableCell>
                  <TableCell className="max-w-xs truncate font-mono text-xs">
                    {row.mini_app_url ?? ''}
                  </TableCell>
                  <TableCell>{displayTimestamp(row.updated_at)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {error && hasSnapshot ? portalsPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
