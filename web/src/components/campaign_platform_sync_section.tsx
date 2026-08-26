import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { to } from '../lib/to.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { formatMoney, ParseDecimal } from '../helpers/money.js';
import {
  deletePlatformCampaignLink,
  fetchPlatformCampaignLinks,
  isPlatformCampaignLicenseError,
  pausePlatformCampaign,
  PLATFORM_CAMPAIGN_NETWORKS,
  refreshPlatformCampaignLink,
  resumePlatformCampaign,
  runPlatformCampaignSync,
  setPlatformCampaignDailyBudget,
  upsertPlatformCampaignLink,
  type PlatformCampaignLink,
  type PlatformCampaignNetwork,
} from '../helpers/platform_campaign_api.js';
import { Button } from './button.js';
import { ErrorBlock } from './error_block.js';

export type CampaignPlatformSyncSectionProps = {
  campaignId: string;
  customerId: string;
  canWrite: boolean;
};

type NetworkFormState = {
  externalCampaignId: string;
  accountId: string;
  budgetUsd: string;
};

const NETWORK_LABELS: Record<PlatformCampaignNetwork, string> = {
  facebook: 'Meta (Facebook)',
  google: 'Google Ads',
};

/**
 * Build default empty form state for platform link fields.
 */
function emptyNetworkForm(): NetworkFormState {
  return { externalCampaignId: '', accountId: '', budgetUsd: '' };
}

/**
 * Map API link row into editable form defaults.
 */
function formFromLink(link: PlatformCampaignLink | undefined): NetworkFormState {
  if (!link) return emptyNetworkForm();
  const budgetUsd =
    link.external_daily_budget_micro != null
      ? (link.external_daily_budget_micro / 1_000_000).toFixed(2)
      : '';
  return {
    externalCampaignId: link.external_campaign_id,
    accountId: link.account_id ?? '',
    budgetUsd,
  };
}

type NetworkPanelProps = {
  network: PlatformCampaignNetwork;
  campaignId: string;
  customerId: string;
  canWrite: boolean;
  link: PlatformCampaignLink | undefined;
  onChanged: () => void;
};

function NetworkPanel({
  network,
  campaignId,
  customerId,
  canWrite,
  link,
  onChanged,
}: NetworkPanelProps) {
  const [form, setForm] = useState<NetworkFormState>(() => formFromLink(link));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setForm(formFromLink(link));
  }, [link]);

  const saveLink = async () => {
    if (!canWrite || busy) return;
    setBusy(true);
    setError(null);
    const [, err] = await to(
      upsertPlatformCampaignLink(campaignId, network, {
        customer_id: customerId,
        external_campaign_id: form.externalCampaignId.trim(),
        account_id: form.accountId.trim() || undefined,
      })
    );
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      setError(mapServiceError(err).message);
      return;
    }
    pushToastMessage({ title: 'Link saved', message: NETWORK_LABELS[network] });
    onChanged();
  };

  const removeLink = async () => {
    if (!canWrite || busy || !link) return;
    setBusy(true);
    setError(null);
    const [, err] = await to(deletePlatformCampaignLink(campaignId, network));
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      setError(mapServiceError(err).message);
      return;
    }
    pushToastMessage({ title: 'Link removed', message: NETWORK_LABELS[network] });
    onChanged();
  };

  const refresh = async () => {
    if (!canWrite || busy || !link) return;
    setBusy(true);
    setError(null);
    const [, err] = await to(refreshPlatformCampaignLink(campaignId, network));
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      setError(mapServiceError(err).message);
      return;
    }
    pushToastMessage({ title: 'Synced', message: NETWORK_LABELS[network] });
    onChanged();
  };

  const pause = async () => {
    if (!canWrite || busy || !link) return;
    setBusy(true);
    setError(null);
    const [, err] = await to(pausePlatformCampaign(campaignId, network));
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      setError(mapServiceError(err).message);
      return;
    }
    pushToastMessage({ title: 'Pause queued', message: NETWORK_LABELS[network] });
    onChanged();
  };

  const resume = async () => {
    if (!canWrite || busy || !link) return;
    setBusy(true);
    setError(null);
    const [, err] = await to(resumePlatformCampaign(campaignId, network));
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      setError(mapServiceError(err).message);
      return;
    }
    pushToastMessage({ title: 'Resume queued', message: NETWORK_LABELS[network] });
    onChanged();
  };

  const applyBudget = async () => {
    if (!canWrite || busy || !link) return;
    setBusy(true);
    setError(null);
    let micro = 0;
    try {
      micro = ParseDecimal(form.budgetUsd.trim());
    } catch {
      setBusy(false);
      setError('Enter a valid daily budget in USD');
      return;
    }
    if (micro <= 0) {
      setBusy(false);
      setError('Daily budget must be greater than zero');
      return;
    }
    const [, err] = await to(setPlatformCampaignDailyBudget(campaignId, network, micro));
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      setError(mapServiceError(err).message);
      return;
    }
    pushToastMessage({ title: 'Budget update queued', message: NETWORK_LABELS[network] });
    onChanged();
  };

  return (
    <div className="card mb-4">
      <h3 className="card__title">{NETWORK_LABELS[network]}</h3>
      <p className="text-muted text-sm mb-3">
        Reuses Cost Sync OAuth tokens for this network. Configure credentials under{' '}
        <Link to="/integrations/cost-sync">Cost Sync</Link> first.
      </p>
      {link ? (
        <dl className="definition-list mb-3">
          <dt>External status</dt>
          <dd className="font-mono">{link.external_status || '-'}</dd>
          <dt>Daily budget (synced)</dt>
          <dd className="font-mono">
            {link.external_daily_budget_micro != null
              ? formatMoney(link.external_daily_budget_micro)
              : '-'}
          </dd>
          <dt>Last synced</dt>
          <dd className="font-mono">
            {link.last_synced_at ? new Date(link.last_synced_at).toLocaleString() : '-'}
          </dd>
          {link.sync_error ? (
            <>
              <dt>Sync error</dt>
              <dd className="text-danger">{link.sync_error}</dd>
            </>
          ) : null}
        </dl>
      ) : null}
      <div className="form-grid mb-3">
        <label className="form-field">
          <span className="form-field__label">External campaign ID</span>
          <input
            className="form-field__input font-mono"
            value={form.externalCampaignId}
            onChange={(e) => setForm((f) => ({ ...f, externalCampaignId: e.target.value }))}
            disabled={!canWrite || busy}
            placeholder={network === 'google' ? '12345678901' : '120000000000000000'}
          />
        </label>
        <label className="form-field">
          <span className="form-field__label">
            {network === 'google' ? 'Google Ads customer ID' : 'Ad account ID (optional)'}
          </span>
          <input
            className="form-field__input font-mono"
            value={form.accountId}
            onChange={(e) => setForm((f) => ({ ...f, accountId: e.target.value }))}
            disabled={!canWrite || busy}
            placeholder={network === 'google' ? '1234567890' : 'act_123456789'}
          />
        </label>
        {link ? (
          <label className="form-field">
            <span className="form-field__label">New daily budget (USD)</span>
            <input
              className="form-field__input font-mono"
              value={form.budgetUsd}
              onChange={(e) => setForm((f) => ({ ...f, budgetUsd: e.target.value }))}
              disabled={!canWrite || busy}
              placeholder="100.00"
            />
          </label>
        ) : null}
      </div>
      {error ? <p className="text-danger text-sm mb-2">{error}</p> : null}
      <div className="button-row flex-wrap">
        {canWrite ? (
          <Button
            label={link ? 'Update link' : 'Save link'}
            variant="primary"
            disabled={busy}
            onClick={() => void saveLink()}
          />
        ) : null}
        {canWrite && link ? (
          <>
            <Button
              label="Refresh status"
              variant="secondary"
              disabled={busy}
              onClick={() => void refresh()}
            />
            <Button
              label="Pause on platform"
              variant="secondary"
              disabled={busy}
              onClick={() => void pause()}
            />
            <Button
              label="Resume on platform"
              variant="secondary"
              disabled={busy}
              onClick={() => void resume()}
            />
            <Button
              label="Set daily budget"
              variant="secondary"
              disabled={busy}
              onClick={() => void applyBudget()}
            />
            <Button
              label="Remove link"
              variant="danger"
              disabled={busy}
              onClick={() => void removeLink()}
            />
          </>
        ) : null}
      </div>
    </div>
  );
}

export function CampaignPlatformSyncSection({
  campaignId,
  customerId,
  canWrite,
}: CampaignPlatformSyncSectionProps) {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [licenseBlocked, setLicenseBlocked] = useState(false);
  const [links, setLinks] = useState<PlatformCampaignLink[]>([]);
  const [syncBusy, setSyncBusy] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    setLicenseBlocked(false);
    const [rows, err] = await to(fetchPlatformCampaignLinks(campaignId));
    setLoading(false);
    if (err) {
      if (isPlatformCampaignLicenseError(err)) {
        setLicenseBlocked(true);
        return;
      }
      setError(err);
      return;
    }
    setLinks(rows ?? []);
  }, [campaignId]);

  useEffect(() => {
    void load();
  }, [load]);

  const syncAll = async () => {
    if (!canWrite || syncBusy) return;
    setSyncBusy(true);
    const [, err] = await to(runPlatformCampaignSync(campaignId));
    setSyncBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Sync failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({
      title: 'Sync started',
      message: 'Platform worker refreshed linked campaigns',
    });
    void load();
  };

  const linkFor = (network: PlatformCampaignNetwork) =>
    links.find((row) => row.network === network);

  if (loading) {
    return <p className="text-muted">Loading platform campaign links...</p>;
  }

  if (licenseBlocked) {
    return (
      <div className="stub-banner">
        <p className="stub-banner__message">
          Platform campaign API (Meta/Google pause, resume, and budget sync) requires an Enterprise
          license (`ad_platform_campaign_api`). Upgrade JWT on{' '}
          <Link to="/settings/license">Settings - License</Link>.
        </p>
      </div>
    );
  }

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Platform sync unavailable" />;
  }

  return (
    <div>
      <p className="text-muted mb-3">
        Link ad-event-processor campaigns to Meta or Google Ads for read-only status sync and
        idempotent pause, resume, or daily budget mutations. Does not create creatives or ad sets.
      </p>
      {canWrite ? (
        <div className="mb-4">
          <Button
            label="Run sync for this campaign"
            variant="secondary"
            disabled={syncBusy}
            onClick={() => void syncAll()}
          />
        </div>
      ) : null}
      {PLATFORM_CAMPAIGN_NETWORKS.map((network) => (
        <NetworkPanel
          key={network}
          network={network}
          campaignId={campaignId}
          customerId={customerId}
          canWrite={canWrite}
          link={linkFor(network)}
          onChanged={() => void load()}
        />
      ))}
    </div>
  );
}
