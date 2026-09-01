import { useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { toast } from 'sonner';

import {
  configureTelegramBot,
  createTelegramDeeplink,
  getTelegramBot,
  getTelegramDeeplink,
  validateTelegramInitData,
} from '@/api/telegram_api';
import { getCampaign } from '@/api/campaigns_api';
import { useBreadcrumbSegmentLabel } from '@/components/system/breadcrumb_context';
import { TelegramBotEditor } from '@/domains/telegram/bot_editor';
import { useResource } from '@/hooks/use_resource';

function parseAuthDateTtl(value: string): number | undefined {
  const trimmed = value.trim();
  if (!trimmed) {
    return undefined;
  }
  const parsed = Number(trimmed);
  return Number.isFinite(parsed) ? parsed : undefined;
}

export function TelegramBotEditorPage() {
  const { campaignId: routeCampaignId } = useParams<{ campaignId: string }>();
  const campaignId = routeCampaignId ?? '';
  const [reloadToken, setReloadToken] = useState(0);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!campaignId) {
        return Promise.resolve(undefined);
      }
      return getTelegramBot(campaignId, signal);
    },
    [campaignId, reloadToken],
  );

  const { data: campaign } = useResource(
    (signal) => {
      if (!campaignId) {
        return Promise.resolve(undefined);
      }
      return getCampaign(campaignId, signal);
    },
    [campaignId],
  );

  useBreadcrumbSegmentLabel(campaignId || undefined, campaign?.name);

  const bot = data;
  const [draftWebhookUrl, setDraftWebhookUrl] = useState('');
  const [draftMiniAppUrl, setDraftMiniAppUrl] = useState('');
  const [draftSecretToken, setDraftSecretToken] = useState('');
  const [draftAuthDateTtl, setDraftAuthDateTtl] = useState('');
  const [draftInitData, setDraftInitData] = useState('');
  const [draftDeeplinkToken, setDraftDeeplinkToken] = useState('');
  const [validateResult, setValidateResult] = useState<
    Awaited<ReturnType<typeof validateTelegramInitData>> | undefined
  >(undefined);
  const [deeplinkResult, setDeeplinkResult] = useState<
    Awaited<ReturnType<typeof createTelegramDeeplink>> | undefined
  >(undefined);
  const [acting, setActing] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>(undefined);
  const [actionMessage, setActionMessage] = useState<string | undefined>(undefined);

  const [draftBotToken, setDraftBotToken] = useState('');

  useEffect(() => {
    if (!bot) {
      return;
    }
    setDraftBotToken(bot.bot_token ?? '');
    setDraftWebhookUrl(bot.webhook_url ?? '');
    setDraftMiniAppUrl(bot.mini_app_url ?? '');
    setDraftSecretToken(bot.secret_token ?? '');
    setDraftAuthDateTtl(bot.auth_date_ttl != null ? String(bot.auth_date_ttl) : '');
  }, [bot]);

  const bumpReload = useCallback(() => {
    setReloadToken((value) => value + 1);
  }, []);

  const onSaveBot = useCallback(() => {
    if (!campaignId) {
      return;
    }
    setActing(true);
    setActionError(undefined);
    setActionMessage(undefined);
    const authDateTtl = parseAuthDateTtl(draftAuthDateTtl);
    void configureTelegramBot(campaignId, {
      campaign_id: campaignId,
      bot_token: draftBotToken,
      webhook_url: draftWebhookUrl,
      mini_app_url: draftMiniAppUrl,
      secret_token: draftSecretToken,
      auth_date_ttl: authDateTtl,
      bot_id: bot?.bot_id,
    })
      .then(() => {
        setActionMessage('Bot configuration saved');
        toast.success('Bot configuration saved');
        bumpReload();
        return getTelegramBot(campaignId);
      })
      .then((next) => {
        if (next) {
          setDraftBotToken(next.bot_token ?? '');
          setDraftWebhookUrl(next.webhook_url ?? '');
          setDraftMiniAppUrl(next.mini_app_url ?? '');
          setDraftSecretToken(next.secret_token ?? '');
          setDraftAuthDateTtl(next.auth_date_ttl != null ? String(next.auth_date_ttl) : '');
        }
      })
      .catch((err: unknown) => {
        setActionError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setActing(false);
      });
  }, [
    bot?.bot_id,
    bumpReload,
    campaignId,
    draftAuthDateTtl,
    draftBotToken,
    draftMiniAppUrl,
    draftSecretToken,
    draftWebhookUrl,
  ]);

  const onValidateInitData = useCallback(() => {
    if (!campaignId) {
      return;
    }
    setActing(true);
    setActionError(undefined);
    setActionMessage(undefined);
    void validateTelegramInitData({
      campaign_id: campaignId,
      init_data: draftInitData,
    })
      .then((result) => {
        setValidateResult(result);
        setActionMessage('initData validated');
        toast.success('initData validated');
      })
      .catch((err: unknown) => {
        setActionError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setActing(false);
      });
  }, [campaignId, draftInitData]);

  const onCreateDeeplink = useCallback(() => {
    if (!campaignId) {
      return;
    }
    setActing(true);
    setActionError(undefined);
    setActionMessage(undefined);
    void createTelegramDeeplink({
      campaign_id: campaignId,
      token: '',
      expires_at: new Date().toISOString(),
    })
      .then((result) => {
        setDeeplinkResult(result);
        setDraftDeeplinkToken(result.token ?? '');
        setActionMessage('Deeplink created');
        toast.success('Deeplink created');
      })
      .catch((err: unknown) => {
        setActionError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setActing(false);
      });
  }, [campaignId]);

  const onResolveDeeplink = useCallback(() => {
    const token = draftDeeplinkToken.trim();
    if (!token) {
      setActionError(new Error('Deeplink token is required'));
      return;
    }
    setActing(true);
    setActionError(undefined);
    setActionMessage(undefined);
    void getTelegramDeeplink(token)
      .then((result) => {
        setDeeplinkResult(result);
        setActionMessage('Deeplink resolved');
        toast.success('Deeplink resolved');
      })
      .catch((err: unknown) => {
        setActionError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setActing(false);
      });
  }, [draftDeeplinkToken]);

  return (
    <TelegramBotEditor
      campaignId={campaignId}
      bot={bot}
      fetching={fetching}
      error={error}
      hasSnapshot={!campaignId || data != null || Boolean(error)}
      draftBotToken={draftBotToken}
      draftWebhookUrl={draftWebhookUrl}
      draftMiniAppUrl={draftMiniAppUrl}
      draftSecretToken={draftSecretToken}
      draftAuthDateTtl={draftAuthDateTtl}
      draftInitData={draftInitData}
      draftDeeplinkToken={draftDeeplinkToken}
      validateResult={validateResult}
      deeplinkResult={deeplinkResult}
      acting={acting}
      actionError={actionError}
      actionMessage={actionMessage}
      onDraftBotTokenChange={setDraftBotToken}
      onDraftWebhookUrlChange={setDraftWebhookUrl}
      onDraftMiniAppUrlChange={setDraftMiniAppUrl}
      onDraftSecretTokenChange={setDraftSecretToken}
      onDraftAuthDateTtlChange={setDraftAuthDateTtl}
      onDraftInitDataChange={setDraftInitData}
      onDraftDeeplinkTokenChange={setDraftDeeplinkToken}
      onSaveBot={onSaveBot}
      onValidateInitData={onValidateInitData}
      onCreateDeeplink={onCreateDeeplink}
      onResolveDeeplink={onResolveDeeplink}
    />
  );
}
