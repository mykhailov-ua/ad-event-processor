import { useCallback, useEffect, useState } from 'react';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import { formatMicro } from '../helpers/money.js';
import {
  createMarginGuardPolicy,
  fetchMarginGuardActivity,
  fetchMarginGuardPolicies,
  removeMarginGuardOverride,
} from '../helpers/margin_guard_api.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { Button } from './button.js';
import { Checkbox } from './checkbox.js';

type MarginGuardPolicy = {
  name?: string;
  min_clicks?: number;
  roi_floor_pct?: number;
  cost_over_revenue_threshold_bps?: number;
  is_active?: boolean;
};

type MarginGuardActivity = {
  created_at?: string;
  placement_id?: string;
  action?: string;
  reason?: string;
};

type MarginSnapshot = {
  advertiser_spend_micro?: number;
  rtb_cost_micro?: number;
  operator_margin_micro?: number;
  publisher_payout_micro?: number;
  threshold_bps?: number;
  margin_breach?: boolean;
};

export type CampaignMarginGuardSectionProps = {
  campaignId: string;
  canWrite: boolean;
};

function TableSkeleton({ cols, rows = 5 }: { cols: number; rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }, (_, i) => (
        <tr key={`mg-skel-${i}`} className="data-table__row--skeleton" aria-hidden="true">
          {Array.from({ length: cols }, (__, j) => (
            <td key={`mg-skel-${i}-${j}`}>
              <span className="skeleton-bar" />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}

export function CampaignMarginGuardSection({
  campaignId,
  canWrite,
}: CampaignMarginGuardSectionProps) {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [policies, setPolicies] = useState<MarginGuardPolicy[]>([]);
  const [activity, setActivity] = useState<MarginGuardActivity[]>([]);
  const [margin, setMargin] = useState<MarginSnapshot | null>(null);
  const [form, setForm] = useState({
    name: 'Default policy',
    min_clicks: '100',
    roi_floor_pct: '0',
    zero_conv_streak: '3',
    cost_over_revenue_threshold_bps: '500',
    is_active: true,
  });
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    const [polRes, actRes, marginRes] = await Promise.all([
      fetchMarginGuardPolicies(campaignId),
      fetchMarginGuardActivity(campaignId),
      to(api(`/api/v1/campaigns/${campaignId}/margin`)),
    ]);
    setLoading(false);
    setPolicies((polRes ?? []) as MarginGuardPolicy[]);
    setActivity((actRes ?? []) as MarginGuardActivity[]);
    setMargin((marginRes[0]?.data as MarginSnapshot | null | undefined) ?? null);
  }, [campaignId]);

  useEffect(() => {
    void load();
  }, [load]);

  const savePolicy = async () => {
    if (!canWrite || saving) return;
    setSaving(true);
    setError(null);
    const body = {
      campaign_id: campaignId,
      name: form.name.trim() || 'Policy',
      min_clicks: Number.parseInt(form.min_clicks, 10) || 0,
      roi_floor_pct: Number.parseFloat(form.roi_floor_pct) || 0,
      zero_conv_streak: Number.parseInt(form.zero_conv_streak, 10) || 0,
      cost_over_revenue_threshold_bps:
        Number.parseInt(form.cost_over_revenue_threshold_bps, 10) || 500,
      is_active: form.is_active,
    };
    const [, err] = await to(createMarginGuardPolicy(body));
    setSaving(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      setError(mapServiceError(err).message);
      return;
    }
    pushToastMessage({ title: 'Policy saved', message: body.name });
    void load();
  };

  const clearOverride = async (placementId: string) => {
    const [, err] = await to(removeMarginGuardOverride(campaignId, placementId));
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Override failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({ title: 'Override removed', message: placementId });
    void load();
  };

  return (
    <div className="stack">
      {loading ? <p className="text-muted">Loading margin guard…</p> : null}
      {error ? <p className="text-danger text-sm">{error}</p> : null}
      {margin ? (
        <div className="metric-row section-block">
          <div className="metric-card">
            <div className="metric-card__label">1h spend</div>
            <div className="metric-card__value font-mono">
              ${formatMicro(margin.advertiser_spend_micro ?? 0)}
            </div>
          </div>
          <div className="metric-card">
            <div className="metric-card__label">RTB cost</div>
            <div className="metric-card__value font-mono">
              ${formatMicro(margin.rtb_cost_micro ?? 0)}
            </div>
          </div>
          <div className="metric-card">
            <div className="metric-card__label">Operator margin</div>
            <div className="metric-card__value font-mono">
              ${formatMicro(margin.operator_margin_micro ?? 0)}
            </div>
          </div>
          <div className="metric-card">
            <div className="metric-card__label">Publisher payout</div>
            <div className="metric-card__value font-mono">
              ${formatMicro(margin.publisher_payout_micro ?? 0)}
            </div>
          </div>
          <div className="metric-card">
            <div className="metric-card__label">Threshold (bps)</div>
            <div className="metric-card__value">{String(margin.threshold_bps ?? '—')}</div>
          </div>
          <div className="metric-card">
            <div className="metric-card__label">Breach</div>
            <div className={`metric-card__value${margin.margin_breach ? ' text-danger' : ''}`}>
              {margin.margin_breach ? 'Yes' : 'No'}
            </div>
          </div>
        </div>
      ) : null}
      <div className="section-card stack">
        <h3 className="subsection-title">Policies</h3>
        <div className="table-wrapper">
          <table className="data-table" aria-label="Margin guard policies">
            <thead>
              <tr>
                <th scope="col">Name</th>
                <th scope="col">Min clicks</th>
                <th scope="col">ROI floor %</th>
                <th scope="col">Cost/rev bps</th>
                <th scope="col">Active</th>
              </tr>
            </thead>
            <tbody>
              {loading ? <TableSkeleton cols={5} /> : null}
              {!loading && policies.length === 0 ? (
                <tr>
                  <td colSpan={5}>No policies yet.</td>
                </tr>
              ) : null}
              {policies.map((p, idx) => (
                <tr key={`${p.name ?? ''}-${idx}`}>
                  <td>{p.name ?? '—'}</td>
                  <td>{String(p.min_clicks ?? 0)}</td>
                  <td>{String(p.roi_floor_pct ?? 0)}</td>
                  <td>{String(p.cost_over_revenue_threshold_bps ?? 0)}</td>
                  <td>{p.is_active ? 'Yes' : 'No'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {canWrite ? (
          <div className="stack mt-4">
            <h4 className="subsection-title">Add policy</h4>
            <label className="form-field" htmlFor="mg-name">
              Name
              <input
                id="mg-name"
                className="form-input form-input--sm"
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              />
            </label>
            <div className="form-row">
              <label className="form-field" htmlFor="mg-min-clicks">
                Min clicks
                <input
                  id="mg-min-clicks"
                  className="form-input form-input--sm"
                  inputMode="numeric"
                  value={form.min_clicks}
                  onChange={(e) => setForm((f) => ({ ...f, min_clicks: e.target.value }))}
                />
              </label>
              <label className="form-field" htmlFor="mg-roi">
                ROI floor %
                <input
                  id="mg-roi"
                  className="form-input form-input--sm"
                  inputMode="decimal"
                  value={form.roi_floor_pct}
                  onChange={(e) => setForm((f) => ({ ...f, roi_floor_pct: e.target.value }))}
                />
              </label>
            </div>
            <div className="form-row">
              <label className="form-field" htmlFor="mg-streak">
                Zero-conv streak
                <input
                  id="mg-streak"
                  className="form-input form-input--sm"
                  inputMode="numeric"
                  value={form.zero_conv_streak}
                  onChange={(e) => setForm((f) => ({ ...f, zero_conv_streak: e.target.value }))}
                />
              </label>
              <label className="form-field" htmlFor="mg-bps">
                Cost over revenue (bps)
                <input
                  id="mg-bps"
                  className="form-input form-input--sm"
                  inputMode="numeric"
                  value={form.cost_over_revenue_threshold_bps}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, cost_over_revenue_threshold_bps: e.target.value }))
                  }
                />
              </label>
            </div>
            <Checkbox
              label="Active"
              checked={form.is_active}
              onChange={(checked) => setForm((f) => ({ ...f, is_active: checked }))}
            />
            <Button
              label={saving ? 'Saving…' : 'Create policy'}
              variant="primary"
              size="sm"
              loading={saving}
              disabled={saving}
              onClick={() => void savePolicy()}
            />
          </div>
        ) : null}
      </div>
      <div className="section-card stack">
        <h3 className="subsection-title">Activity</h3>
        <div className="table-wrapper">
          <table className="data-table" aria-label="Margin guard activity">
            <thead>
              <tr>
                <th scope="col">Time</th>
                <th scope="col">Placement</th>
                <th scope="col">Action</th>
                <th scope="col">Reason</th>
                <th scope="col" />
              </tr>
            </thead>
            <tbody>
              {loading ? <TableSkeleton cols={5} /> : null}
              {!loading && activity.length === 0 ? (
                <tr>
                  <td colSpan={5}>
                    <div className="empty-state">
                      <div className="empty-state__title">No guard actions yet</div>
                      <div className="empty-state__desc text-muted text-sm">
                        Margin guard activity appears after policies trigger.
                      </div>
                    </div>
                  </td>
                </tr>
              ) : null}
              {activity.map((row, idx) => (
                <tr key={`${row.placement_id ?? ''}-${row.created_at ?? ''}-${idx}`}>
                  <td>{row.created_at ? new Date(row.created_at).toLocaleString() : '—'}</td>
                  <td className="font-mono text-hint">{row.placement_id ?? '—'}</td>
                  <td>{row.action ?? '—'}</td>
                  <td>{row.reason ?? '—'}</td>
                  <td>
                    {canWrite && row.action === 'pause' && row.placement_id ? (
                      <Button
                        label="Remove override"
                        variant="secondary"
                        size="sm"
                        onClick={() => void clearOverride(row.placement_id!)}
                      />
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
