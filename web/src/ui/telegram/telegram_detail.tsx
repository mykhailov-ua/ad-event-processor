import { useEffect, useState } from 'react';
import {
  configureTelegramBot,
  createDeeplinkToken,
  createTelegramPostback,
  fetchTelegramBots,
  fetchTelegramPostbacks,
  TELEGRAM_DETAIL_TABS,
  type TelegramBot,
  type TelegramDetailTab,
  type TelegramPostback,
} from '../../helpers/telegram_api.js';
import { ConfirmCancelledError } from '../../helpers/confirmed_api.js';
import { pushToastMessage } from '../../helpers/toast_ui.js';
import { ContextBar } from '../shell/context_bar.js';
import { PageChrome } from '../system/page_chrome.js';
import { TabBar } from '../system/tab_bar.js';
import { Button } from '../system/button.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import { StubBanner } from '../system/stub_banner.js';
import styles from './telegram_detail.module.css';

export type TelegramDetailProps = {
  campaignId: string;
  tab: TelegramDetailTab;
  maskedOnly: boolean;
  onTabChange: (tab: TelegramDetailTab) => void;
};

export function TelegramDetailView({
  campaignId,
  tab,
  maskedOnly,
  onTabChange,
}: TelegramDetailProps) {
  const [bots, setBots] = useState<TelegramBot[]>([]);
  const [botsLoading, setBotsLoading] = useState(false);
  const [botsError, setBotsError] = useState<unknown>(null);

  const [postbacks, setPostbacks] = useState<TelegramPostback[]>([]);
  const [postbacksLoading, setPostbacksLoading] = useState(false);
  const [postbacksError, setPostbacksError] = useState<unknown>(null);

  const [postbackUrl, setPostbackUrl] = useState('');
  const [creatingPostback, setCreatingPostback] = useState(false);
  const [postbackActionError, setPostbackActionError] = useState<unknown>(null);

  const [deeplinkPayload, setDeeplinkPayload] = useState('{}');
  const [deeplinkResult, setDeeplinkResult] = useState<string | null>(null);
  const [deeplinkError, setDeeplinkError] = useState<unknown>(null);
  const [creatingDeeplink, setCreatingDeeplink] = useState(false);

  useEffect(() => {
    if (tab !== 'bots') return;
    setBotsLoading(true);
    setBotsError(null);
    void fetchTelegramBots()
      .then((items) => setBots(items.filter((b) => !b.campaign_id || b.campaign_id === campaignId)))
      .catch((err) => setBotsError(err))
      .finally(() => setBotsLoading(false));
  }, [tab, campaignId]);

  useEffect(() => {
    if (tab !== 'postbacks') return;
    setPostbacksLoading(true);
    setPostbacksError(null);
    void fetchTelegramPostbacks(campaignId)
      .then(setPostbacks)
      .catch((err) => setPostbacksError(err))
      .finally(() => setPostbacksLoading(false));
  }, [tab, campaignId]);

  const onCreatePostback = async () => {
    setCreatingPostback(true);
    setPostbackActionError(null);
    try {
      await createTelegramPostback({ campaign_id: campaignId, postback_url: postbackUrl });
      pushToastMessage({ title: 'Postback created', message: 'Telegram postback saved' });
      const items = await fetchTelegramPostbacks(campaignId);
      setPostbacks(items);
      setPostbackUrl('');
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      setPostbackActionError(err);
    } finally {
      setCreatingPostback(false);
    }
  };

  const onCreateDeeplink = async () => {
    setCreatingDeeplink(true);
    setDeeplinkError(null);
    setDeeplinkResult(null);
    let body: Record<string, string> = { campaign_id: campaignId };
    try {
      const parsed = JSON.parse(deeplinkPayload) as Record<string, string>;
      body = { ...parsed, campaign_id: campaignId };
    } catch {
      body = { campaign_id: campaignId };
    }
    try {
      const result = await createDeeplinkToken(body);
      setDeeplinkResult(JSON.stringify(result, null, 2));
      pushToastMessage({ title: 'Deeplink created', message: 'Token generated' });
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      setDeeplinkError(err);
    } finally {
      setCreatingDeeplink(false);
    }
  };

  return (
    <div className={styles.root}>
      <ContextBar parentLabel="Campaigns" parentTo="/campaigns" currentLabel="Telegram" />
      <PageChrome title="Telegram integration" badge={<span className={styles.mono}>{campaignId}</span>} />
      <TabBar
        tabs={TELEGRAM_DETAIL_TABS}
        active={tab}
        onChange={(next) => onTabChange(next as TelegramDetailTab)}
      />
      <div className={styles.panel} role="tabpanel">
        {tab === 'bots' ? (
          <div className={styles.panel}>
            {botsLoading ? <PageSkeleton rows={4} /> : null}
            {botsError ? <ErrorBlock error={botsError} fallbackTitle="Failed to load bots" /> : null}
            <div className={styles.table}>
              <div className={styles.tableHead}>
                <span>Bot ID</span>
                <span>Webhook</span>
                <span>Mini app</span>
              </div>
              {bots.map((bot) => (
                <div key={bot.id ?? bot.bot_id} className={styles.tableRow}>
                  <span className={styles.mono}>{bot.bot_id ?? bot.id ?? '-'}</span>
                  <span className={styles.mono}>{bot.webhook_url ?? '-'}</span>
                  <span className={styles.mono}>{bot.mini_app_url ?? '-'}</span>
                </div>
              ))}
            </div>
            {bots[0]?.bot_id ? (
              <div className={styles.actions}>
                <Button
                  variant="secondary"
                 
                  type="button"
                  onClick={() =>
                    void configureTelegramBot(bots[0].bot_id ?? '', { campaign_id: campaignId }).then(() =>
                      pushToastMessage({ title: 'Bot linked', message: 'Campaign linked to bot' })
                    )
                  }
                >
                  Link first bot to campaign
                </Button>
              </div>
            ) : null}
          </div>
        ) : null}
        {tab === 'postbacks' ? (
          <div className={styles.panel}>
            {postbacksLoading ? <PageSkeleton rows={3} /> : null}
            {postbacksError ? <ErrorBlock error={postbacksError} fallbackTitle="Failed to load postbacks" /> : null}
            {postbackActionError ? (
              <ErrorBlock error={postbackActionError} fallbackTitle="Create postback failed" />
            ) : null}
            <div className={styles.table}>
              <div className={styles.tableHead}>
                <span>ID</span>
                <span>URL</span>
                <span />
              </div>
              {postbacks.map((row) => (
                <div key={row.id} className={styles.tableRow}>
                  <span className={styles.mono}>{row.id ?? '-'}</span>
                  <span className={styles.mono}>{row.postback_url ?? '-'}</span>
                  <span />
                </div>
              ))}
            </div>
            <form
              className={styles.form}
              onSubmit={(e) => {
                e.preventDefault();
                void onCreatePostback();
              }}
            >
              <label className={styles.field}>
                <span className={styles.label}>Postback URL</span>
                <input
                  className={styles.input}
                  required
                  value={postbackUrl}
                  onChange={(e) => setPostbackUrl(e.target.value)}
                />
              </label>
              <Button type="submit" variant="primary" disabled={creatingPostback || !postbackUrl.trim()}>
                {creatingPostback ? 'Creating...' : 'Add postback'}
              </Button>
            </form>
          </div>
        ) : null}
        {tab === 'deeplink' ? (
          <div className={styles.panel}>
            {deeplinkError ? <ErrorBlock error={deeplinkError} fallbackTitle="Deeplink failed" /> : null}
            <form
              className={styles.form}
              onSubmit={(e) => {
                e.preventDefault();
                void onCreateDeeplink();
              }}
            >
              <label className={styles.field}>
                <span className={styles.label}>Extra fields (JSON)</span>
                <textarea
                  className={styles.input}
                  rows={4}
                  value={deeplinkPayload}
                  onChange={(e) => setDeeplinkPayload(e.target.value)}
                />
              </label>
              <Button type="submit" variant="primary" disabled={creatingDeeplink}>
                {creatingDeeplink ? 'Creating...' : 'Create deeplink token'}
              </Button>
            </form>
            {deeplinkResult ? <pre className={styles.pre}>{deeplinkResult}</pre> : null}
          </div>
        ) : null}
        {tab === 'reports' ? (
          <div className={styles.panel}>
            {maskedOnly ? (
              <StubBanner
                title="Reports unavailable"
                message="Telegram reports require full campaigns:read permission."
              />
            ) : (
              <StubBanner
                title="Reports not wired"
                message="Open the TG reports hub from the reports directory when available."
              />
            )}
          </div>
        ) : null}
      </div>
    </div>
  );
}
