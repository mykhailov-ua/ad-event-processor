import { Link } from 'react-router-dom';

import { FilterPanel } from '@/shell/filter_panel';
import { PageChrome } from '@/shell/page_chrome';
import { EmptyState } from '@/shell/empty_state';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
  directoryTableRevalidatingClass,
} from '@/shell/directory_table';
import type { TelegramBot } from '@/api/types';
import { TelegramNav, telegramPanelError } from '@/domains/telegram/telegram_nav';
import { displayTimestamp } from '@/lib/display';

export type TelegramBotsDirectoryProps = {
  bots?: TelegramBot[];
  fetching: boolean;
  listRevalidating?: boolean;
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
  listRevalidating = false,
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
      <PageChrome title="Telegram bots" controlPanel={<TelegramNav />}>
        {telegramPanelError(error, 'Could not load Telegram bots')}
      </PageChrome>
    );
  }

  return (
    <PageChrome
      controlPanel={
        <div className="grid gap-3">
          <TelegramNav />
          <FilterPanel className="w-full max-w-xl gap-4">
            <h2 className="text-base font-semibold">Configure bot</h2>
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
          </FilterPanel>
        </div>
      }
      title="Telegram bots"
    >
      {(bots ?? []).length === 0 ? (
        <EmptyState title="No bots" description="No Telegram Mini App bots are configured." />
      ) : (
        <DirectoryTable
          className={directoryTableRevalidatingClass(listRevalidating)}
          horizontalScroll
        >
          <TableHeader>
            <TableRow>
              <DirectoryTableHead>Bot ID</DirectoryTableHead>
              <DirectoryTableHead>Campaign</DirectoryTableHead>
              <DirectoryTableHead>Webhook</DirectoryTableHead>
              <DirectoryTableHead>Mini App</DirectoryTableHead>
              <DirectoryTableHead>Updated</DirectoryTableHead>
              <DirectoryTableHead />
            </TableRow>
          </TableHeader>
          <TableBody>
            {(bots ?? []).map((row) => {
              const campaignId = row.campaign_id ?? '';
              return (
                <TableRow key={String(row.bot_id ?? campaignId ?? row.webhook_url)}>
                  <TableCell>{row.bot_id ?? ''}</TableCell>
                  <TableCell className="text-xs">{campaignId}</TableCell>
                  <TableCell className="whitespace-nowrap text-xs">
                    {row.webhook_url ?? ''}
                  </TableCell>
                  <TableCell className="whitespace-nowrap text-xs">
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
        </DirectoryTable>
      )}

      {actionError ? telegramPanelError(actionError, 'Bot action failed') : null}
      {error && hasSnapshot ? telegramPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
