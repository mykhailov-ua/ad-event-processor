import { useCallback, useEffect, useState } from 'react';
import { to } from '../lib/to.js';
import {
  createBrand,
  createBrandCreative,
  deleteBrandCreative,
  fetchBrandCreatives,
  updateBrandCreative,
} from '../helpers/brand_creatives_api.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { Button } from './button.js';
import { StatusBadge } from './status_badge.js';

export type BrandCreativeRow = {
  id: string;
  name: string;
  landing_url: string;
  weight: number;
  status: string;
};

export type CampaignBrandCreativesSectionProps = {
  brandId: string;
  customerId: string;
  canWrite: boolean;
  onBrandCreated: (id: string) => void;
};

function asCreative(raw: unknown): BrandCreativeRow | null {
  if (!raw || typeof raw !== 'object') return null;
  const o = raw as Record<string, unknown>;
  if (typeof o.id !== 'string') return null;
  return {
    id: o.id,
    name: typeof o.name === 'string' ? o.name : '',
    landing_url: typeof o.landing_url === 'string' ? o.landing_url : '',
    weight: typeof o.weight === 'number' ? o.weight : Number(o.weight) || 0,
    status: typeof o.status === 'string' ? o.status : 'ACTIVE',
  };
}

function TableSkeleton({ cols, rows = 5 }: { cols: number; rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }, (_, i) => (
        <tr key={`cr-skel-${i}`} className="data-table__row--skeleton" aria-hidden="true">
          {Array.from({ length: cols }, (__, j) => (
            <td key={`cr-skel-${i}-${j}`}>
              <span className="skeleton-bar" />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}

export function CampaignBrandCreativesSection({
  brandId: initialBrandId,
  customerId,
  canWrite,
  onBrandCreated,
}: CampaignBrandCreativesSectionProps) {
  const [brandId, setBrandId] = useState(initialBrandId);
  const [creatives, setCreatives] = useState<BrandCreativeRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [outboxHint, setOutboxHint] = useState<string | null>(null);
  const [form, setForm] = useState({
    name: '',
    landing_url: '',
    weight: '100',
    status: 'ACTIVE',
  });

  useEffect(() => {
    setBrandId(initialBrandId);
  }, [initialBrandId]);

  const markOutboxQueued = (action: string) => {
    setOutboxHint(
      `${action} queued to outbox — live on tracker typically within 60s (Redis brand creatives sync).`
    );
  };

  const load = useCallback(async () => {
    if (!brandId) {
      setLoading(false);
      setCreatives([]);
      return;
    }
    setLoading(true);
    const [rows, err] = await to(fetchBrandCreatives(brandId));
    setLoading(false);
    setCreatives(
      err ? [] : (rows ?? []).map(asCreative).filter((c): c is BrandCreativeRow => c != null)
    );
  }, [brandId]);

  useEffect(() => {
    void load();
  }, [load]);

  const ensureBrand = async (): Promise<string | null> => {
    if (brandId) return brandId;
    if (!canWrite || !customerId) return null;
    const [id, err] = await to(createBrand(customerId, 'Default brand'));
    if (err) {
      pushToastMessage({ title: 'Brand create failed', message: mapServiceError(err).message });
      return null;
    }
    const nextId = id ?? '';
    setBrandId(nextId);
    onBrandCreated(nextId);
    return nextId || null;
  };

  const addCreative = async () => {
    if (!canWrite || saving) return;
    setSaving(true);
    const bid = await ensureBrand();
    if (!bid) {
      setSaving(false);
      return;
    }
    const body = {
      name: form.name.trim() || 'Landing',
      landing_url: form.landing_url.trim(),
      weight: Number.parseInt(form.weight, 10) || 100,
      status: form.status,
    };
    const [, err] = await to(createBrandCreative(bid, body));
    setSaving(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Creative save failed', message: mapServiceError(err).message });
      return;
    }
    markOutboxQueued('Creative create');
    pushToastMessage({ title: 'Creative saved', message: 'Hot-path sync queued via outbox' });
    setForm({ name: '', landing_url: '', weight: '100', status: 'ACTIVE' });
    void load();
  };

  const togglePause = async (creative: BrandCreativeRow) => {
    if (!canWrite) return;
    const next = creative.status === 'PAUSED' ? 'ACTIVE' : 'PAUSED';
    const [, err] = await to(
      updateBrandCreative(creative.id, {
        name: creative.name,
        landing_url: creative.landing_url,
        weight: creative.weight,
        status: next,
      })
    );
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Update failed', message: mapServiceError(err).message });
      return;
    }
    markOutboxQueued(next === 'PAUSED' ? 'Creative pause' : 'Creative resume');
    void load();
  };

  const removeCreative = async (id: string) => {
    if (!canWrite) return;
    const [, err] = await to(deleteBrandCreative(id));
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Delete failed', message: mapServiceError(err).message });
      return;
    }
    markOutboxQueued('Creative delete');
    void load();
  };

  return (
    <div className="stack" data-testid="campaign-brand-creatives">
      <p className="text-muted text-sm">
        Weighted landing URLs for click redirect rotation. Changes sync to Redis via outbox.
      </p>
      {outboxHint ? (
        <p className="text-sm" data-testid="creative-outbox-sync">
          <StatusBadge status="pending" kind="service" label="Outbox sync" /> {outboxHint}
        </p>
      ) : null}
      {!brandId && !canWrite ? (
        <p className="text-muted">No brand linked to this campaign.</p>
      ) : null}
      {brandId ? (
        <p className="text-hint text-sm">
          Brand: <span className="font-mono">{brandId}</span>
        </p>
      ) : null}
      <div className="table-wrapper">
        <table className="data-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Landing URL</th>
              <th>Weight</th>
              <th>Status</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {loading ? <TableSkeleton cols={5} /> : null}
            {!loading && creatives.length === 0 ? (
              <tr>
                <td colSpan={5}>
                  <div className="empty-state">
                    <div className="empty-state__title">No creatives yet</div>
                    <div className="empty-state__desc text-muted text-sm">
                      Add a landing URL and weight to start A/B rotation.
                    </div>
                  </div>
                </td>
              </tr>
            ) : null}
            {creatives.map((c) => (
              <tr key={c.id}>
                <td>{c.name}</td>
                <td className="font-mono text-hint text-sm">{c.landing_url}</td>
                <td>{String(c.weight)}</td>
                <td>
                  <StatusBadge status={c.status === 'ACTIVE' ? 'ACTIVE' : 'PAUSED'} />
                </td>
                <td>
                  {canWrite ? (
                    <div className="flex gap-2">
                      <Button
                        label={c.status === 'PAUSED' ? 'Resume' : 'Pause'}
                        variant="secondary"
                        size="sm"
                        onClick={() => void togglePause(c)}
                      />
                      <Button
                        label="Delete"
                        variant="secondary"
                        size="sm"
                        onClick={() => void removeCreative(c.id)}
                      />
                    </div>
                  ) : null}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {canWrite ? (
        <div className="section-card stack mt-4">
          <h3 className="subsection-title">Add creative</h3>
          <label className="form-field">
            Name
            <input
              className="form-input form-input--sm"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
            />
          </label>
          <label className="form-field">
            Landing URL
            <input
              className="form-input"
              type="url"
              value={form.landing_url}
              onChange={(e) => setForm((f) => ({ ...f, landing_url: e.target.value }))}
            />
          </label>
          <div className="form-row">
            <label className="form-field">
              Weight
              <input
                className="form-input form-input--sm"
                inputMode="numeric"
                value={form.weight}
                onChange={(e) => setForm((f) => ({ ...f, weight: e.target.value }))}
              />
            </label>
            <label className="form-field">
              Status
              <select
                className="form-select"
                value={form.status}
                onChange={(e) => setForm((f) => ({ ...f, status: e.target.value }))}
              >
                <option value="ACTIVE">Active</option>
                <option value="PAUSED">Paused</option>
              </select>
            </label>
          </div>
          <Button
            label={brandId ? 'Add creative' : 'Create brand & add'}
            variant="primary"
            size="sm"
            loading={saving}
            disabled={saving || !form.landing_url.trim()}
            onClick={() => void addCreative()}
          />
        </div>
      ) : null}
    </div>
  );
}
