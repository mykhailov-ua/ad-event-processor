import { Link } from 'react-router-dom';

import { PageChrome } from '@/components/system/page_chrome';
import { EmptyState } from '@/components/system/empty_state';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { Button } from '@/components/ui/button';
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
import type { TelegramBot } from '@/api/types';
import { TelegramNav, telegramPanelError } from '@/domains/telegram/telegram_nav';
import { displayTimestamp } from '@/lib/display';

export type TelegramBotsDirectoryProps = {
  bots: TelegramBot[];
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  draftCampaignId: string;
  acting: boolean;
  actionError: Error | undefined;
  onDraftCampaignIdChange: (value: string) => void;
  onOpenEditor: () => void;
};

export function TelegramBotsDirectory({
  bots,
  fetching,
  error,
  hasSnapshot,
  draftCampaignId,
  acting,
  actionError,
  onDraftCampaignIdChange,
  onOpenEditor,
}: TelegramBotsDirectoryProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Telegram bots">
        <TelegramNav />
        {telegramPanelError(error, 'Could not load Telegram bots')}
      </PageChrome>
    );
  }

  return (
    <PageChrome title="Telegram bots">
      <TelegramNav />

      <section className="ui-filter-panel gap-6">
        <h2 className="text-base font-semibold">Configure bot</h2>
        <div className="grid max-w-xl gap-4">
          <div className="grid gap-2">
            <Label htmlFor="telegram-campaign-id">Campaign ID</Label>
            <Input
              id="telegram-campaign-id"
              placeholder="UUID"
              value={draftCampaignId}
              onChange={(event) => onDraftCampaignIdChange(event.target.value)}
            />
          </div>
          <Button disabled={acting} onClick={onOpenEditor} type="button">
            Open editor
          </Button>
        </div>
      </section>

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
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {bots.map((row) => {
                const campaignId = row.campaign_id ?? '';
                return (
                  <TableRow key={String(row.bot_id ?? campaignId ?? row.webhook_url)}>
                    <TableCell>{row.bot_id ?? ''}</TableCell>
                    <TableCell className="font-mono text-xs">{campaignId}</TableCell>
                    <TableCell className="max-w-xs truncate font-mono text-xs">
                      {row.webhook_url ?? ''}
                    </TableCell>
                    <TableCell className="max-w-xs truncate font-mono text-xs">
                      {row.mini_app_url ?? ''}
                    </TableCell>
                    <TableCell>{displayTimestamp(row.updated_at)}</TableCell>
                    <TableCell>
                      {campaignId ? (
                        <Button asChild type="button" variant="outline">
                          <Link to={`/telegram/bots/${campaignId}`}>Edit</Link>
                        </Button>
                      ) : null}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}

      {actionError ? telegramPanelError(actionError, 'Bot action failed') : null}
      {error && hasSnapshot ? telegramPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
