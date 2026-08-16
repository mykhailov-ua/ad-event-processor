import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { to } from '../lib/to.js';
import { createInFlightGuard } from '../lib/async_guard.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import {
  createTelegramDeeplink,
  createTelegramPostback,
  deleteTelegramPostback,
  fetchTelegramBot,
  fetchTelegramPostbacks,
  saveTelegramBot,
  testTelegramPostback,
  type TelegramDeeplinkDTO,
  type TelegramPostbackDTO,
} from '../helpers/tg_admin_api.js';
import { Button } from './button.js';
import { ErrorBlock } from './error_block.js';
import { SectionCard } from './section_card.js';

export type CampaignTelegramSectionProps = {
  campaignId: string;
  canWrite: boolean;
};

type TelegramBotForm = {
  bot_id: string;
  webhook_url: string;
  mini_app_url: string;
  auth_date_ttl: string;
};

function maskSecret(value: string): string {
  if (!value) return '—';
  if (value.length <= 4) return '••••';
  return `••••${value.slice(-4)}`;
}

async function copyText(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text);
    pushToastMessage({ title: 'Copied', message: 'Copied to clipboard' });
  } catch {
    pushToastMessage({ title: 'Copy failed', message: 'Could not access clipboard' });
  }
}

/**
 * Telegram Mini App configuration for a campaign.
 */
export function CampaignTelegramSection({ campaignId, canWrite }: CampaignTelegramSectionProps) {
  const gateRef = useRef(createInFlightGuard());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [configured, setConfigured] = useState(false);
  const [bot, setBot] = useState<TelegramBotForm>({
    bot_id: '',
    webhook_url: '',
    mini_app_url: '',
    auth_date_ttl: '300',
  });
  const [storedToken, setStoredToken] = useState('');
  const [storedSecret, setStoredSecret] = useState('');
  const [tokenInput, setTokenInput] = useState('');
  const [secretInput, setSecretInput] = useState('');
  const [postbacks, setPostbacks] = useState<TelegramPostbackDTO[]>([]);
  const [newPostbackUrl, setNewPostbackUrl] = useState('');
  const [deeplink, setDeeplink] = useState<TelegramDeeplinkDTO | null>(null);
  const [utmSource, setUtmSource] = useState('');
  const [utmCampaign, setUtmCampaign] = useState('');
  const [saving, setSaving] = useState(false);
  const [postbackBusy, setPostbackBusy] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    const [botRes, postbacksRes] = await Promise.all([
      to(fetchTelegramBot(campaignId)),
      to(fetchTelegramPostbacks(campaignId)),
    ]);
    setLoading(false);
    if (botRes[1]) {
      setError(botRes[1]);
      return;
    }
    if (postbacksRes[1]) {
      setError(postbacksRes[1]);
      return;
    }
    const botData = botRes[0];
    if (botData) {
      setConfigured(true);
      setStoredToken(botData.bot_token ?? '');
      setStoredSecret(botData.secret_token ?? '');
      setBot({
        bot_id: String(botData.bot_id ?? ''),
        webhook_url: botData.webhook_url ?? '',
        mini_app_url: botData.mini_app_url ?? '',
        auth_date_ttl: String(botData.auth_date_ttl ?? 300),
      });
    } else {
      setConfigured(false);
      setStoredToken('');
      setStoredSecret('');
    }
    setPostbacks(postbacksRes[0] ?? []);
  }, [campaignId]);

  useEffect(() => {
    void load();
  }, [load]);

  const webhookIngestUrl = () => {
    const botId = bot.bot_id.trim();
    if (!botId) return '';
    return `${window.location.origin}/api/v1/telegram/webhook/${botId}`;
  };

  const clickBridgeUrl = (token: string) => {
    const q = new URLSearchParams({
      campaign_id: campaignId,
      bridge_token: token,
    });
    return `${window.location.origin}/tg/click?${q.toString()}`;
  };

  const handleSaveBot = async (e: FormEvent) => {
    e.preventDefault();
    if (!canWrite || !gateRef.current.tryAcquire()) return;
    setSaving(true);
    const token = tokenInput.trim() || storedToken;
    const secret = secretInput.trim() || storedSecret;
    const [, err] = await to(saveTelegramBot(campaignId, {
      bot_id: Number(bot.bot_id) || 0,
      bot_token: token,
      webhook_url: bot.webhook_url,
      mini_app_url: bot.mini_app_url,
      secret_token: secret,
      auth_date_ttl: Number(bot.auth_date_ttl) || 300,
    }));
    setSaving(false);
    gateRef.current.release();
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      const mapped = mapServiceError(err);
      pushToastMessage({ title: mapped.title, message: mapped.message, code: mapped.code });
      return;
    }
    setTokenInput('');
    setSecretInput('');
    pushToastMessage({ title: 'Saved', message: 'Telegram bot configuration updated' });
    void load();
  };

  const handleCreateDeeplink = async () => {
    if (!canWrite || !gateRef.current.tryAcquire()) return;
    const [dl, err] = await to(createTelegramDeeplink(campaignId, {
      utm_source: utmSource,
      utm_campaign: utmCampaign,
    }));
    gateRef.current.release();
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      const mapped = mapServiceError(err);
      pushToastMessage({ title: mapped.title, message: mapped.message, code: mapped.code });
      return;
    }
    setDeeplink(dl);
    pushToastMessage({ title: 'Deeplink created', message: 'Bridge token generated' });
  };

  const handleAddPostback = async (e: FormEvent) => {
    e.preventDefault();
    const url = newPostbackUrl.trim();
    if (!url || !canWrite || !gateRef.current.tryAcquire()) return;
    setPostbackBusy(true);
    const [, err] = await to(createTelegramPostback(campaignId, url));
    setPostbackBusy(false);
    gateRef.current.release();
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      const mapped = mapServiceError(err);
      pushToastMessage({ title: mapped.title, message: mapped.message, code: mapped.code });
      return;
    }
    setNewPostbackUrl('');
    pushToastMessage({ title: 'Postback added', message: 'S2S postback URL saved' });
    void load();
  };

  const handleTestPostback = async (id: string | number) => {
    if (!canWrite || !gateRef.current.tryAcquire()) return;
    const [, err] = await to(testTelegramPostback(String(id)));
    gateRef.current.release();
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      const mapped = mapServiceError(err);
      pushToastMessage({ title: mapped.title, message: mapped.message, code: mapped.code });
      return;
    }
    pushToastMessage({ title: 'Test sent', message: 'Postback URL responded OK' });
  };

  const handleDeletePostback = async (id: string | number) => {
    if (!canWrite || !gateRef.current.tryAcquire()) return;
    const [, err] = await to(deleteTelegramPostback(String(id)));
    gateRef.current.release();
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      const mapped = mapServiceError(err);
      pushToastMessage({ title: mapped.title, message: mapped.message, code: mapped.code });
      return;
    }
    pushToastMessage({ title: 'Deleted', message: 'Postback removed' });
    void load();
  };

  if (loading) {
    return <p className="text-muted">Loading Telegram settings…</p>;
  }

  if (error) {
    return <ErrorBlock error={error} />;
  }

  const ingestUrl = webhookIngestUrl();

  return (
    <div className="tg-panel stack stack--lg">
      <div className="page-header page-header--compact">
        <h2 className="page-header__title">Telegram Mini App</h2>
        <p className="page-header__desc text-sm">
          <Link to={`/reports/telegram?campaign_id=${encodeURIComponent(campaignId)}`}>
            Open full analytics
          </Link>
          {' · '}
          Tracking endpoints: <code className="font-mono">/tg/click</code>,{' '}
          <code className="font-mono">/tg/impression</code>
        </p>
      </div>

      <SectionCard
        title="Bot & webhook"
        desc="Configure Telegram Bot API credentials and webhook target."
      >
        <form className="stack" onSubmit={(e) => void handleSaveBot(e)}>
          <label className="form-field" htmlFor="tg-bot-id">
            Bot ID
            <input
              id="tg-bot-id"
              className="form-input"
              inputMode="numeric"
              value={bot.bot_id}
              disabled={!canWrite}
              placeholder="Telegram bot numeric ID"
              onChange={(e) => setBot((b) => ({ ...b, bot_id: e.target.value }))}
            />
          </label>
          <label className="form-field" htmlFor="tg-bot-token">
            {configured ? `Bot token (saved: ${maskSecret(storedToken)})` : 'Bot token'}
            <input
              id="tg-bot-token"
              className="form-input"
              type="password"
              autoComplete="off"
              disabled={!canWrite}
              placeholder={configured ? 'Leave blank to keep current' : 'From @BotFather'}
              value={tokenInput}
              onChange={(e) => setTokenInput(e.target.value)}
            />
          </label>
          <label className="form-field" htmlFor="tg-webhook-url">
            Webhook URL
            <span className="form-hint text-muted text-sm">Public URL registered with Telegram via setWebhook</span>
            <input
              id="tg-webhook-url"
              className="form-input"
              disabled={!canWrite}
              placeholder="https://your-domain/api/v1/telegram/webhook/…"
              value={bot.webhook_url}
              onChange={(e) => setBot((b) => ({ ...b, webhook_url: e.target.value }))}
            />
          </label>
          {ingestUrl ? (
            <div className="toolbar-row text-sm">
              <span className="text-muted">BidShard ingest URL:</span>
              <code className="code-inline flex-1">{ingestUrl}</code>
              <Button label="Copy" variant="secondary" size="sm" onClick={() => void copyText(ingestUrl)} />
            </div>
          ) : null}
          <label className="form-field" htmlFor="tg-secret-token">
            {configured ? `Secret token (saved: ${maskSecret(storedSecret)})` : 'Secret token'}
            <div className="flex items-center gap-2">
              <input
                id="tg-secret-token"
                className="form-input flex-1"
                type="password"
                autoComplete="off"
                disabled={!canWrite}
                placeholder={configured ? 'Leave blank to keep current' : 'X-Telegram-Bot-Api-Secret-Token'}
                value={secretInput}
                onChange={(e) => setSecretInput(e.target.value)}
              />
              {canWrite ? (
                <Button
                  label="Generate"
                  variant="secondary"
                  size="sm"
                  onClick={() => setSecretInput(crypto.randomUUID().replace(/-/g, ''))}
                />
              ) : null}
            </div>
          </label>
          <label className="form-field" htmlFor="tg-mini-app-url">
            Mini App URL
            <input
              id="tg-mini-app-url"
              className="form-input"
              disabled={!canWrite}
              placeholder="https://t.me/your_bot/your_app"
              value={bot.mini_app_url}
              onChange={(e) => setBot((b) => ({ ...b, mini_app_url: e.target.value }))}
            />
          </label>
          <label className="form-field" htmlFor="tg-auth-ttl">
            Session validation window
            <span className="form-hint text-muted text-sm">How long Telegram login data stays valid, in seconds</span>
            <input
              id="tg-auth-ttl"
              className="form-input"
              inputMode="numeric"
              disabled={!canWrite}
              value={bot.auth_date_ttl}
              onChange={(e) => setBot((b) => ({ ...b, auth_date_ttl: e.target.value }))}
            />
          </label>
          {canWrite ? (
            <Button
              label={saving ? 'Saving…' : 'Save bot config'}
              variant="primary"
              type="submit"
              loading={saving}
              disabled={saving}
            />
          ) : null}
        </form>
      </SectionCard>

      <SectionCard
        title="Deeplink / bridge token"
        desc="Create trackable links for Telegram click attribution."
      >
        <div className="stack">
          <label className="form-field" htmlFor="tg-utm-source">
            UTM source (optional)
            <input
              id="tg-utm-source"
              className="form-input"
              disabled={!canWrite}
              value={utmSource}
              onChange={(e) => setUtmSource(e.target.value)}
            />
          </label>
          <label className="form-field" htmlFor="tg-utm-campaign">
            UTM campaign (optional)
            <input
              id="tg-utm-campaign"
              className="form-input"
              disabled={!canWrite}
              value={utmCampaign}
              onChange={(e) => setUtmCampaign(e.target.value)}
            />
          </label>
          {canWrite ? (
            <Button
              label="Generate bridge token"
              variant="secondary"
              size="sm"
              onClick={() => void handleCreateDeeplink()}
            />
          ) : null}
          {deeplink?.token ? (
            <div className="stack stack--sm text-sm">
              <p>
                <strong>Token: </strong>
                <code className="font-mono">{deeplink.token}</code>
              </p>
              <p className="text-muted">
                Expires:{' '}
                {deeplink.expires_at
                  ? new Date(deeplink.expires_at).toLocaleString()
                  : '—'}
              </p>
              <div className="toolbar-row">
                <code className="code-inline flex-1">{clickBridgeUrl(deeplink.token)}</code>
                <Button
                  label="Copy click URL"
                  variant="secondary"
                  size="sm"
                  onClick={() => void copyText(clickBridgeUrl(deeplink.token!))}
                />
              </div>
            </div>
          ) : null}
        </div>
      </SectionCard>

      <SectionCard
        title="S2S postbacks"
        desc="Advertiser conversion callbacks (supports {click_id}, {campaign_id} macros)."
      >
        <div className="stack">
          {postbacks.length > 0 ? (
            <div className="table-wrapper">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>URL</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {postbacks.map((row) => (
                    <tr key={String(row.id)}>
                      <td><code className="code-inline">{row.postback_url}</code></td>
                      <td>
                        {canWrite ? (
                          <div className="cluster--actions">
                            <Button
                              label="Test"
                              variant="secondary"
                              size="sm"
                              onClick={() => void handleTestPostback(String(row.id))}
                            />
                            <Button
                              label="Delete"
                              variant="danger"
                              size="sm"
                              onClick={() => void handleDeletePostback(String(row.id))}
                            />
                          </div>
                        ) : null}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <div className="empty-state">
              <div className="empty-state__title">No postbacks configured</div>
              <div className="empty-state__desc text-muted text-sm">
                Add an S2S postback URL below.
              </div>
            </div>
          )}
          {canWrite ? (
            <form className="stack" onSubmit={(e) => void handleAddPostback(e)}>
              <label className="form-field" htmlFor="tg-new-postback">
                Postback URL
                <input
                  id="tg-new-postback"
                  className="form-input min-w-80"
                  placeholder="https://partner.example/postback?cid={click_id}"
                  value={newPostbackUrl}
                  onChange={(e) => setNewPostbackUrl(e.target.value)}
                />
              </label>
              <Button
                label={postbackBusy ? 'Adding…' : 'Add postback'}
                variant="primary"
                size="sm"
                type="submit"
                loading={postbackBusy}
                disabled={postbackBusy}
              />
            </form>
          ) : null}
        </div>
      </SectionCard>
    </div>
  );
}
