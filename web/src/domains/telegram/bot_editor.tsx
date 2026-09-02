import { Link } from 'react-router-dom';
import { useEffect, useState } from 'react';

import { PageChrome } from '@/shell/page_chrome';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { Textarea } from '@/components/ui/textarea';
import type { TelegramBot, TelegramDeeplink, TelegramValidateResult } from '@/api/types';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import { TelegramNav, telegramPanelError } from '@/domains/telegram/telegram_nav';

export type TelegramBotEditorProps = {
  campaignId: string;
  bot: TelegramBot | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  draftBotToken: string;
  draftWebhookUrl: string;
  draftMiniAppUrl: string;
  draftSecretToken: string;
  draftAuthDateTtl: string;
  draftInitData: string;
  draftDeeplinkToken: string;
  validateResult: TelegramValidateResult | undefined;
  deeplinkResult: TelegramDeeplink | undefined;
  acting: boolean;
  actionError: Error | undefined;
  actionMessage: string | undefined;
  onDraftBotTokenChange: (value: string) => void;
  onDraftWebhookUrlChange: (value: string) => void;
  onDraftMiniAppUrlChange: (value: string) => void;
  onDraftSecretTokenChange: (value: string) => void;
  onDraftAuthDateTtlChange: (value: string) => void;
  onDraftInitDataChange: (value: string) => void;
  onDraftDeeplinkTokenChange: (value: string) => void;
  onSaveBot: () => void;
  onValidateInitData: () => void;
  onCreateDeeplink: () => void;
  onResolveDeeplink: () => void;
};

export function TelegramBotEditor({
  campaignId,
  bot,
  fetching,
  error,
  hasSnapshot,
  draftBotToken,
  draftWebhookUrl,
  draftMiniAppUrl,
  draftSecretToken,
  draftAuthDateTtl,
  draftInitData,
  draftDeeplinkToken,
  validateResult,
  deeplinkResult,
  acting,
  actionError,
  actionMessage,
  onDraftBotTokenChange,
  onDraftWebhookUrlChange,
  onDraftMiniAppUrlChange,
  onDraftSecretTokenChange,
  onDraftAuthDateTtlChange,
  onDraftInitDataChange,
  onDraftDeeplinkTokenChange,
  onSaveBot,
  onValidateInitData,
  onCreateDeeplink,
  onResolveDeeplink,
}: TelegramBotEditorProps) {
  const [botOpen, setBotOpen] = useState(false);
  const [validateOpen, setValidateOpen] = useState(false);
  const [resolveOpen, setResolveOpen] = useState(false);

  useEffect(() => {
    if (actionMessage === 'Bot configuration saved') {
      setBotOpen(false);
    }
    if (actionMessage === 'initData validated') {
      setValidateOpen(false);
    }
    if (actionMessage === 'Deeplink resolved') {
      setResolveOpen(false);
    }
  }, [actionMessage]);

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Telegram bot editor">
        <TelegramNav />
        {telegramPanelError(error, 'Could not load bot config')}
      </PageChrome>
    );
  }

  return (
    <PageChrome title="Telegram bot editor">
      <TelegramNav />
      <Link className="text-sm text-muted-foreground hover:underline" to="/telegram/bots">
        Back to bots
      </Link>
      <p className="text-sm text-muted-foreground font-mono">Campaign {campaignId}</p>
      {bot?.bot_id != null ? (
        <p className="text-sm text-muted-foreground">Bot ID: {bot.bot_id}</p>
      ) : null}

      <div className="flex flex-wrap gap-2">
        <Button onClick={() => setBotOpen(true)} type="button">
          Configure bot
        </Button>
        <Button onClick={() => setValidateOpen(true)} type="button" variant="outline">
          Validate initData
        </Button>
        <Button disabled={acting} onClick={onCreateDeeplink} type="button" variant="outline">
          Create deeplink
        </Button>
        <Button onClick={() => setResolveOpen(true)} type="button" variant="outline">
          Resolve deeplink
        </Button>
      </div>

      <Sheet onOpenChange={setBotOpen} open={botOpen}>
        <SheetContent className="overflow-y-auto sm:max-w-xl">
          <SheetHeader>
            <SheetTitle>Bot configuration</SheetTitle>
          </SheetHeader>
          <div className="grid gap-4 pt-4">
            <div className="grid gap-2">
              <Label htmlFor="telegram-bot-token">Bot token</Label>
              <Input
                id="telegram-bot-token"
                type="password"
                value={draftBotToken}
                onChange={(event) => onDraftBotTokenChange(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="telegram-webhook-url">Webhook URL</Label>
              <Input
                id="telegram-webhook-url"
                value={draftWebhookUrl}
                onChange={(event) => onDraftWebhookUrlChange(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="telegram-mini-app-url">Mini App URL</Label>
              <Input
                id="telegram-mini-app-url"
                value={draftMiniAppUrl}
                onChange={(event) => onDraftMiniAppUrlChange(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="telegram-secret-token">Secret token</Label>
              <Input
                id="telegram-secret-token"
                type="password"
                value={draftSecretToken}
                onChange={(event) => onDraftSecretTokenChange(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="telegram-auth-date-ttl">Auth date TTL (seconds)</Label>
              <Input
                id="telegram-auth-date-ttl"
                value={draftAuthDateTtl}
                onChange={(event) => onDraftAuthDateTtlChange(event.target.value)}
              />
            </div>
            <Button disabled={acting} onClick={onSaveBot} type="button">
              Save bot
            </Button>
          </div>
        </SheetContent>
      </Sheet>

      <Dialog onOpenChange={setValidateOpen} open={validateOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Validate initData</DialogTitle>
          </DialogHeader>
          <div className="grid gap-2">
            <Label htmlFor="telegram-init-data">initData</Label>
            <Textarea
              id="telegram-init-data"
              rows={4}
              value={draftInitData}
              onChange={(event) => onDraftInitDataChange(event.target.value)}
            />
          </div>
          {validateResult ? (
            <JsonDashboardView payload={validateResult as unknown as Record<string, unknown>} />
          ) : null}
          <DialogFooter>
            <Button disabled={acting} onClick={onValidateInitData} type="button">
              Validate
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog onOpenChange={setResolveOpen} open={resolveOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Resolve deeplink</DialogTitle>
          </DialogHeader>
          <div className="grid gap-2">
            <Label htmlFor="telegram-deeplink-token">Token</Label>
            <Input
              id="telegram-deeplink-token"
              value={draftDeeplinkToken}
              onChange={(event) => onDraftDeeplinkTokenChange(event.target.value)}
            />
          </div>
          {deeplinkResult ? (
            <JsonDashboardView payload={deeplinkResult as unknown as Record<string, unknown>} />
          ) : null}
          <DialogFooter>
            <Button disabled={acting} onClick={onResolveDeeplink} type="button">
              Resolve
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {actionMessage ? (
        <p className="text-sm text-muted-foreground">{actionMessage}</p>
      ) : null}
      {actionError ? telegramPanelError(actionError, 'Telegram action failed') : null}
      {error && hasSnapshot ? telegramPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
