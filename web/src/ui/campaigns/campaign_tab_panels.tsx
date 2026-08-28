import { useEffect, useState } from 'react';
import {
  buildCampaignEventsUrl,
  fetchCampaignStats,
  patchCampaign,
  patchCampaignFraud,
  publishCampaign,
  putPostbackConfig,
  runPublishCheck,
  type Campaign,
  type CampaignFraud,
  type CampaignStats,
  type PostbackConfig,
} from '../../helpers/campaigns_api.js';
import { ConfirmCancelledError } from '../../helpers/confirmed_api.js';
import { formatLocaleDateTime as formatDate } from '../../helpers/format_display.js';
import { formatAmountMicro, formatUsdDecimal } from '../../helpers/money.js';
import { pushToastMessage } from '../../helpers/toast_ui.js';
import { useResource } from '../../helpers/use_resource.js';
import { Button } from '../system/button.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import { PaginationBar } from '../system/pagination_bar.js';
import styles from './campaign_detail.module.css';

const EVENTS_PAGE = 25;

export function CampaignToolbar({
  campaignId,
  onReload,
}: {
  campaignId: string;
  onReload: () => void;
}) {
  const [checking, setChecking] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [checkResult, setCheckResult] = useState<string | null>(null);
  const [error, setError] = useState<unknown>(null);

  const onPublishCheck = async () => {
    setChecking(true);
    setError(null);
    try {
      const result = await runPublishCheck(campaignId);
      setCheckResult(JSON.stringify(result, null, 2));
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      setError(err);
    } finally {
      setChecking(false);
    }
  };

  const onPublish = async () => {
    setPublishing(true);
    setError(null);
    try {
      await publishCampaign(campaignId);
      pushToastMessage({ title: 'Published', message: 'Campaign publish requested' });
      onReload();
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      setError(err);
    } finally {
      setPublishing(false);
    }
  };

  return (
    <div className={styles.panel}>
      <div className={styles.toolbar}>
        <Button variant="secondary" type="button" disabled={checking} onClick={() => void onPublishCheck()}>
          {checking ? 'Checking...' : 'Publish check'}
        </Button>
        <Button variant="primary" type="button" disabled={publishing} onClick={() => void onPublish()}>
          {publishing ? 'Publishing...' : 'Publish'}
        </Button>
      </div>
      {error ? <ErrorBlock error={error} fallbackTitle="Publish action failed" /> : null}
      {checkResult ? <pre className={styles.pre}>{checkResult}</pre> : null}
    </div>
  );
}

export function CampaignOverviewPanel({ campaign }: { campaign: Campaign }) {
  return (
    <dl className={styles.dl}>
      <dt>ID</dt>
      <dd className={styles.mono}>{campaign.id ?? '-'}</dd>
      <dt>Name</dt>
      <dd>{campaign.name ?? '-'}</dd>
      <dt>Status</dt>
      <dd>{campaign.status ?? '-'}</dd>
      <dt>Customer</dt>
      <dd className={styles.mono}>{campaign.customer_id ?? '-'}</dd>
      <dt>Budget limit</dt>
      <dd>{formatUsdDecimal(campaign.budget_limit)}</dd>
      <dt>Current spend</dt>
      <dd>{formatUsdDecimal(campaign.current_spend)}</dd>
      <dt>Pacing mode</dt>
      <dd>{campaign.pacing_mode ?? '-'}</dd>
      <dt>Timezone</dt>
      <dd>{campaign.timezone ?? '-'}</dd>
      <dt>Target URL</dt>
      <dd className={styles.mono}>{campaign.target_url ?? '-'}</dd>
      <dt>Updated</dt>
      <dd>{formatDate(campaign.updated_at)}</dd>
    </dl>
  );
}

export function CampaignConfigPanel({
  campaign,
  onSaved,
}: {
  campaign: Campaign;
  onSaved: () => void;
}) {
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [form, setForm] = useState<Record<string, string>>({});

  const values = {
    name: form.name ?? campaign.name ?? '',
    status: form.status ?? campaign.status ?? '',
    budget_limit: form.budget_limit ?? campaign.budget_limit ?? '',
    pacing_mode: form.pacing_mode ?? campaign.pacing_mode ?? '',
    timezone: form.timezone ?? campaign.timezone ?? '',
    target_url: form.target_url ?? campaign.target_url ?? '',
  };

  const onSave = async () => {
    setSaving(true);
    setError(null);
    try {
      await patchCampaign(campaign.id ?? '', {
        name: values.name,
        status: values.status,
        budget_limit: values.budget_limit,
        pacing_mode: values.pacing_mode,
        timezone: values.timezone,
        target_url: values.target_url,
      });
      pushToastMessage({ title: 'Saved', message: 'Campaign configuration updated' });
      onSaved();
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      setError(err);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className={styles.panel}>
      {error ? <ErrorBlock error={error} fallbackTitle="Save failed" /> : null}
      <form
        className={styles.form}
        onSubmit={(e) => {
          e.preventDefault();
          void onSave();
        }}
      >
        <label className={styles.field}>
          <span className={styles.label}>Name</span>
          <input
            className={styles.input}
            value={values.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
          />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>Status</span>
          <input
            className={styles.input}
            value={values.status}
            onChange={(e) => setForm({ ...form, status: e.target.value })}
          />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>Budget limit</span>
          <input
            className={styles.input}
            value={values.budget_limit}
            onChange={(e) => setForm({ ...form, budget_limit: e.target.value })}
          />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>Pacing mode</span>
          <input
            className={styles.input}
            value={values.pacing_mode}
            onChange={(e) => setForm({ ...form, pacing_mode: e.target.value })}
          />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>Timezone</span>
          <input
            className={styles.input}
            value={values.timezone}
            onChange={(e) => setForm({ ...form, timezone: e.target.value })}
          />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>Target URL</span>
          <input
            className={styles.input}
            value={values.target_url}
            onChange={(e) => setForm({ ...form, target_url: e.target.value })}
          />
        </label>
        <div className={styles.actions}>
          <Button type="submit" variant="primary" disabled={saving}>
            {saving ? 'Saving...' : 'Save configuration'}
          </Button>
        </div>
      </form>
    </div>
  );
}

export function CampaignFraudPanel({ campaignId }: { campaignId: string }) {
  const url = `/api/v1/campaigns/${encodeURIComponent(campaignId)}/fraud`;
  const { data, loading, error, reload } = useResource<CampaignFraud>(url);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<unknown>(null);
  const [form, setForm] = useState<Partial<CampaignFraud>>({});

  if (loading && !data) return <PageSkeleton rows={6} />;
  if (error) return <ErrorBlock error={error} fallbackTitle="Failed to load fraud settings" />;

  const fraud = data ?? {};
  const values = {
    silent_reject_enabled:
      form.silent_reject_enabled ?? fraud.silent_reject_enabled ?? false,
    fraud_threshold_pass:
      form.fraud_threshold_pass != null
        ? String(form.fraud_threshold_pass)
        : fraud.fraud_threshold_pass != null
          ? String(fraud.fraud_threshold_pass)
          : '',
    fraud_threshold_suspect:
      form.fraud_threshold_suspect != null
        ? String(form.fraud_threshold_suspect)
        : fraud.fraud_threshold_suspect != null
          ? String(fraud.fraud_threshold_suspect)
          : '',
    fraud_threshold_ivt:
      form.fraud_threshold_ivt != null
        ? String(form.fraud_threshold_ivt)
        : fraud.fraud_threshold_ivt != null
          ? String(fraud.fraud_threshold_ivt)
          : '',
    fraud_threshold_block:
      form.fraud_threshold_block != null
        ? String(form.fraud_threshold_block)
        : fraud.fraud_threshold_block != null
          ? String(fraud.fraud_threshold_block)
          : '',
  };

  const onSave = async () => {
    setSaving(true);
    setSaveError(null);
    const body: CampaignFraud = {
      silent_reject_enabled: values.silent_reject_enabled,
      fraud_threshold_pass: Number.parseFloat(values.fraud_threshold_pass) || 0,
      fraud_threshold_suspect: Number.parseFloat(values.fraud_threshold_suspect) || 0,
      fraud_threshold_ivt: Number.parseFloat(values.fraud_threshold_ivt) || 0,
      fraud_threshold_block: Number.parseFloat(values.fraud_threshold_block) || 0,
    };
    try {
      await patchCampaignFraud(campaignId, body);
      pushToastMessage({ title: 'Saved', message: 'Fraud settings updated' });
      reload();
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      setSaveError(err);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className={styles.panel}>
      {saveError ? <ErrorBlock error={saveError} fallbackTitle="Save failed" /> : null}
      <form
        className={styles.form}
        onSubmit={(e) => {
          e.preventDefault();
          void onSave();
        }}
      >
        <label className={styles.field}>
          <span className={styles.label}>Silent reject enabled</span>
          <input
            type="checkbox"
            checked={values.silent_reject_enabled}
            onChange={(e) => setForm({ ...form, silent_reject_enabled: e.target.checked })}
          />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>Threshold pass</span>
          <input
            className={styles.input}
            inputMode="decimal"
            value={values.fraud_threshold_pass}
            onChange={(e) =>
              setForm({ ...form, fraud_threshold_pass: Number.parseFloat(e.target.value) })
            }
          />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>Threshold suspect</span>
          <input
            className={styles.input}
            inputMode="decimal"
            value={values.fraud_threshold_suspect}
            onChange={(e) =>
              setForm({ ...form, fraud_threshold_suspect: Number.parseFloat(e.target.value) })
            }
          />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>Threshold IVT</span>
          <input
            className={styles.input}
            inputMode="decimal"
            value={values.fraud_threshold_ivt}
            onChange={(e) =>
              setForm({ ...form, fraud_threshold_ivt: Number.parseFloat(e.target.value) })
            }
          />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>Threshold block</span>
          <input
            className={styles.input}
            inputMode="decimal"
            value={values.fraud_threshold_block}
            onChange={(e) =>
              setForm({ ...form, fraud_threshold_block: Number.parseFloat(e.target.value) })
            }
          />
        </label>
        <div className={styles.actions}>
          <Button type="submit" variant="primary" disabled={saving}>
            {saving ? 'Saving...' : 'Save fraud settings'}
          </Button>
        </div>
      </form>
    </div>
  );
}

export function CampaignStatsPanel({ campaignId }: { campaignId: string }) {
  const [stats, setStats] = useState<CampaignStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);

  const load = () => {
    setLoading(true);
    setError(null);
    void fetchCampaignStats(campaignId)
      .then((result) => setStats(result))
      .catch((err) => setError(err))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
  }, [campaignId]);

  if (loading && !stats) return <PageSkeleton rows={4} />;
  if (error) return <ErrorBlock error={error} fallbackTitle="Failed to load statistics" />;

  return (
    <div className={styles.panel}>
      <div className={styles.kpiStrip}>
        <div className={styles.kpiCard}>
          <span className={styles.kpiLabel}>Impressions</span>
          <span className={styles.kpiValue}>{String(stats?.impressions ?? 0)}</span>
        </div>
        <div className={styles.kpiCard}>
          <span className={styles.kpiLabel}>Clicks</span>
          <span className={styles.kpiValue}>{String(stats?.clicks ?? 0)}</span>
        </div>
        <div className={styles.kpiCard}>
          <span className={styles.kpiLabel}>Conversions</span>
          <span className={styles.kpiValue}>{String(stats?.conversions ?? 0)}</span>
        </div>
        <div className={styles.kpiCard}>
          <span className={styles.kpiLabel}>Spend</span>
          <span className={styles.kpiValue}>{formatAmountMicro(stats?.spend_micro)}</span>
        </div>
      </div>
      {stats?.freshness_label ? (
        <p className={styles.kpiLabel}>
          {stats.freshness_label}
          {stats.stale ? ' (stale)' : ''}
        </p>
      ) : null}
      <Button variant="secondary" type="button" onClick={load}>
        Refresh
      </Button>
    </div>
  );
}

export function CampaignEventsPanel({ campaignId }: { campaignId: string }) {
  const [offset, setOffset] = useState(0);
  const url = buildCampaignEventsUrl(campaignId, EVENTS_PAGE, offset);
  const { data, loading, error } = useResource(url);

  if (error) return <ErrorBlock error={error} fallbackTitle="Failed to load events" />;

  const events = data as {
    items?: Array<{ id?: string; event_type?: string; created_at?: string; payout_micro?: number }>;
    total?: number;
  };

  return (
    <div className={styles.panel}>
      {loading && !events.items?.length ? <PageSkeleton rows={5} /> : null}
      <div className={styles.table}>
        <div className={styles.tableHead}>
          <span>Type</span>
          <span>Payout</span>
          <span>Created</span>
          <span>ID</span>
        </div>
        {(events.items ?? []).map((row, index) => (
          <div key={row.id ?? `${row.created_at ?? index}`} className={styles.tableRow}>
            <span>{row.event_type ?? '-'}</span>
            <span>{formatAmountMicro(row.payout_micro)}</span>
            <span>{formatDate(row.created_at)}</span>
            <span className={styles.mono}>{row.id ?? '-'}</span>
          </div>
        ))}
      </div>
      <PaginationBar
        limit={EVENTS_PAGE}
        offset={offset}
        total={events.total ?? 0}
        onOffsetChange={setOffset}
      />
    </div>
  );
}

export function CampaignPostbacksPanel({ campaignId }: { campaignId: string }) {
  const url = `/api/v1/postbacks/config/${encodeURIComponent(campaignId)}`;
  const { data, loading, error, reload } = useResource<PostbackConfig>(url);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<unknown>(null);
  const [form, setForm] = useState<Partial<PostbackConfig>>({});

  if (loading && !data) return <PageSkeleton rows={5} />;
  if (error) return <ErrorBlock error={error} fallbackTitle="Failed to load postback config" />;

  const config = data ?? {};
  const values = {
    provider: form.provider ?? config.provider ?? '',
    url_template: form.url_template ?? config.url_template ?? '',
    api_token: form.api_token ?? config.api_token ?? '',
    target_event: form.target_event ?? config.target_event ?? '',
    test_event_code: form.test_event_code ?? config.test_event_code ?? '',
  };

  const onSave = async () => {
    setSaving(true);
    setSaveError(null);
    const body: PostbackConfig = { ...values };
    try {
      await putPostbackConfig(campaignId, body);
      pushToastMessage({ title: 'Saved', message: 'Postback configuration updated' });
      reload();
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      setSaveError(err);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className={styles.panel}>
      {saveError ? <ErrorBlock error={saveError} fallbackTitle="Save failed" /> : null}
      <form
        className={styles.form}
        onSubmit={(e) => {
          e.preventDefault();
          void onSave();
        }}
      >
        <label className={styles.field}>
          <span className={styles.label}>Provider</span>
          <input
            className={styles.input}
            value={values.provider}
            onChange={(e) => setForm({ ...form, provider: e.target.value })}
          />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>URL template</span>
          <input
            className={styles.input}
            value={values.url_template}
            onChange={(e) => setForm({ ...form, url_template: e.target.value })}
          />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>API token</span>
          <input
            className={styles.input}
            type="password"
            autoComplete="off"
            value={values.api_token}
            onChange={(e) => setForm({ ...form, api_token: e.target.value })}
          />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>Target event</span>
          <input
            className={styles.input}
            value={values.target_event}
            onChange={(e) => setForm({ ...form, target_event: e.target.value })}
          />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>Test event code</span>
          <input
            className={styles.input}
            value={values.test_event_code}
            onChange={(e) => setForm({ ...form, test_event_code: e.target.value })}
          />
        </label>
        <div className={styles.actions}>
          <Button type="submit" variant="primary" disabled={saving}>
            {saving ? 'Saving...' : 'Save postback config'}
          </Button>
        </div>
      </form>
    </div>
  );
}
